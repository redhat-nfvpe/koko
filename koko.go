/**
 * koko: Container connector
 */
package main

import (
	"fmt"
	"github.com/mattn/go-getopt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/containernetworking/cni/pkg/ns"
	"github.com/vishvananda/netlink"

	"github.com/docker/docker/client"
	"golang.org/x/net/context"

	"github.com/MakeNowJust/heredoc"
)

// VERSION indicates koko's version.
var VERSION = "master@git"

func makeVethPair(name, peer string, mtu int) (netlink.Link, error) {
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name:  name,
			Flags: net.FlagUp,
			MTU:   mtu,
		},
		PeerName: peer,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return nil, err
	}
	return veth, nil
}

func getVethPair(name1 string, name2 string) (link1 netlink.Link,
	link2 netlink.Link, err error) {
	link1, err = makeVethPair(name1, name2, 1500)
	if err != nil {
		switch {
		case os.IsExist(err):
			err = fmt.Errorf(
				"container veth name provided (%v) "+
					"already exists", name1)
			return
		default:
			err = fmt.Errorf("failed to make veth pair: %v", err)
			return
		}
	}

	link2, err = netlink.LinkByName(name2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to lookup %q: %v\n", name2, err)
	}

	return
}

// addVxLanInterface creates VxLan interface by given vxlan object
func addVxLanInterface(vxlan vxLan, devName string) error {
	parentIF, err := netlink.LinkByName(vxlan.parentIF)

	if err != nil {
		return fmt.Errorf("Failed to get %s: %v", vxlan.parentIF, err)
	}

	vxlanconf := netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:   devName,
			TxQLen: 1000,
		},
		VxlanId:      vxlan.id,
		VtepDevIndex: parentIF.Attrs().Index,
		Group:        vxlan.ipAddr,
		Port:         4789,
		Learning:     true,
		L2miss:       true,
		L3miss:       true,
	}
	err = netlink.LinkAdd(&vxlanconf)

	if err != nil {
		return fmt.Errorf("Failed to add vxlan %s: %v", devName, err)
	}
	return nil
}

// getDockerContainerNS retrieves container's network namespace from
// docker container id, given as containerID.
func getDockerContainerNS(containerID string) (namespace string, err error) {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	cli.UpdateClientVersion("1.24")

	json, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		err = fmt.Errorf("failed to get container info: %v", err)
		return
	}
	if json.NetworkSettings == nil {
		err = fmt.Errorf("failed to get container info: %v", err)
		return
	}
	namespace = fmt.Sprintf("/proc/%d/ns/net", json.State.Pid)
	return
}

// vEth is a structure to descrive veth interfaces.
type vEth struct {
	nsName   string      // What's the network namespace?
	linkName string      // And what will we call the link.
	ipAddr   []net.IPNet // Slice of IPv4/v6 adress.
}

// vxLan is a structure to descrive vxlan endpoint.
type vxLan struct {
	parentIF string // parent interface name
	id       int    // VxLan ID
	ipAddr   net.IP // VxLan destination address
}

// setVethLink is low-level handler to set IP address onveth links given
// a single vEth data object.
// ...primarily used privately by makeVeth().
func (veth *vEth) setVethLink(link netlink.Link) (err error) {
	vethNs, err := ns.GetNS(veth.nsName)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}
	defer vethNs.Close()

	if err := netlink.LinkSetNsFd(link, int(vethNs.Fd())); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}

	err = vethNs.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(veth.linkName)
		if err != nil {
			return fmt.Errorf("failed to lookup %q in %q: %v",
				veth.linkName, vethNs.Path(), err)
		}

		if err = netlink.LinkSetUp(link); err != nil {
			return fmt.Errorf("failed to set %q up: %v",
				veth.linkName, err)
		}

		// Conditionally set the IP address.
		for i := 0; i < len(veth.ipAddr); i++ {
			addr := &netlink.Addr{IPNet: &veth.ipAddr[i], Label: ""}
			if err = netlink.AddrAdd(link, addr); err != nil {
				return fmt.Errorf(
					"failed to add IP addr %v to %q: %v",
					addr, veth.linkName, err)
			}
		}

		return nil
	})

	return
}

// removeVethLink is low-level handler to get interface handle in
// container/netns namespace and remove it.
func (veth *vEth) removeVethLink() (err error) {
	vethNs, err := ns.GetNS(veth.nsName)

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}
	defer vethNs.Close()

	err = vethNs.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(veth.linkName)
		if err != nil {
			return fmt.Errorf("failed to lookup %q in %q: %v",
				veth.linkName, vethNs.Path(), err)
		}

		err = netlink.LinkDel(link)
		if err != nil {
			return fmt.Errorf("failed to remove link %q in %q: %v",
				veth.linkName, vethNs.Path(), err)
		}
		return nil
	})

	return
}

// makeVeth is top-level handler to create veth links given two vEth data
// objects: veth1 and veth2.
func makeVeth(veth1 vEth, veth2 vEth) {

	link1, link2, err := getVethPair(veth1.linkName, veth2.linkName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}

	veth1.setVethLink(link1)
	veth2.setVethLink(link2)
}

// makeVxLan makes vxlan interface and put it into container namespace
func makeVxLan(veth1 vEth, vxlan vxLan) {

	err := addVxLanInterface(vxlan, veth1.linkName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vxlan add failed: %v", err)
	}

	link, err2 := netlink.LinkByName(veth1.linkName)
	if err2 != nil {
		fmt.Fprintf(os.Stderr, "Cannot get %s: %v", veth1.linkName, err)
	}
	err = veth1.setVethLink(link)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot add IPaddr/netns failed: %v",
			err)
	}
}

func parseLinkIPOption(veth *vEth, n []string) (err error) {

	veth.linkName = n[0]

	numAddr := len(n) - 1

	veth.ipAddr = make([]net.IPNet, numAddr)
	for i := 0; i < numAddr; i++ {
		ip, mask, err1 := net.ParseCIDR(n[i+1])
		if err1 != nil {
			err = fmt.Errorf("failed to parse IP addr(%d) %s: %v",
				i, n[i], err1)
			return
		}
		veth.ipAddr[i].IP = ip
		veth.ipAddr[i].Mask = mask.Mask
	}
	return
}

// parseNOption parses '-n' option and put this information in veth object.
func parseNOption(s string) (veth vEth, err error) {
	n := strings.Split(s, ",")
	if len(n) > 4 || len(n) < 1 {
		err = fmt.Errorf("failed to parse %s", s)
		return
	}

	veth.nsName = fmt.Sprintf("/var/run/netns/%s", n[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	err1 := parseLinkIPOption(&veth, n[1:])
	if err1 != nil {
		fmt.Fprintf(os.Stderr, "%v", err1)
		os.Exit(1)
	}

	return
}

// Parsedoption Parses '-n' option and put this information in veth object.
func parseDOption(s string) (veth vEth, err error) {
	n := strings.Split(s, ",")
	if len(n) > 4 || len(n) < 1 {
		err = fmt.Errorf("failed to parse %s", s)
		return
	}

	veth.nsName, err = getDockerContainerNS(n[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	err1 := parseLinkIPOption(&veth, n[1:])
	if err1 != nil {
		fmt.Fprintf(os.Stderr, "%v", err1)
		os.Exit(1)
	}

	return
}

// parseXOption parses '-x' option and put this information in veth object.
func parseXOption(s string) (vxlan vxLan, err error) {
	var err2 error // if we encounter an error, it's marked here.

	n := strings.Split(s, ",")
	if len(n) != 3 {
		err = fmt.Errorf("failed to parse %s", s)
		return
	}

	vxlan.parentIF = n[0]
	vxlan.ipAddr = net.ParseIP(n[1])
	vxlan.id, err2 = strconv.Atoi(n[2])
	if err2 != nil {
		err = fmt.Errorf("failed to parse VXID %s: %v", n[2], err2)
		return
	}

	return
}

// usage shows usage when user invokes it with '-h' option.
func usage() {
	doc := heredoc.Doc(`
		
		Usage:
		./koko -d centos1,link1,192.168.1.1/24 -d centos2,link2,192.168.1.2/24 #with IP addr
		./koko -d centos1,link1 -d centos2,link2  #without IP addr
		./koko -n /var/run/netns/test1,link1,192.168.1.1/24 <other>	

                See https://github.com/redhat-nfvpe/koko/wiki/Examples for the detail.
	`)

	fmt.Print(doc)

}

/**
Usage:
* case1: connect between docker container, with ip address
./koko -d centos1:link1:192.168.1.1/24 -d centos2:link2:192.168.1.2/24
* case2: connect between docker container, without ip address
./koko -d centos1:link1 -d centos2:link2
* case3: connect between linux ns container (a.k.a. 'ip netns'), with ip address
./koko -n /var/run/netns/test1:link1:192.168.1.1/24 -n <snip>
* case4: connect between linux ns and docker container
./koko -n /var/run/netns/test1:link1:192.168.1.1/24 -d centos2:link2:192.168.1.2/24
* case5: connect docker/linux ns container to vxlan interface
./koko -d centos1:link1:192.168.1.1/24 -x eth1:1.1.1.1:10

* case6: delete docker interface
./koko -D centos1:link1
* case7: delete linux ns interface
./koko -N /var/run/netns/test1:link1
*/
func main() {

	var c int     // command line parameters.
	var err error // if we encounter an error, it's marked here.
	const (
		ModeUnspec = iota
		ModeAddVeth
		ModeAddVxlan
		ModeDeleteLink
	)

	cnt := 0 // Count of command line parameters.
	// Any errors with peeling apart the command line options.
	getopt.OptErr = 0

	// Create some empty vEth data objects.
	veth1 := vEth{}
	veth2 := vEth{}
	vxlan := vxLan{}
	mode := ModeUnspec

	// Parse options and and exit if they don't meet our criteria.
	for {
		if c = getopt.Getopt("d:n:x:hD:v"); c == getopt.EOF {
			break
		}
		switch c {
		case 'd':
			if cnt == 0 {
				veth1, err = parseDOption(getopt.OptArg)
				if err != nil {
					fmt.Fprintf(os.Stderr,
						"Parse failed %s!:%v",
						getopt.OptArg, err)
					usage()
					os.Exit(1)
				}
			} else if cnt == 1 {
				veth2, err = parseDOption(getopt.OptArg)
				if err != nil {
					fmt.Fprintf(os.Stderr,
						"Parse failed %s!:%v",
						getopt.OptArg, err)
					usage()
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Too many config!")
				usage()
				os.Exit(1)
			}
			cnt++

		case 'D':
			if cnt == 0 {
				veth1, err = parseDOption(getopt.OptArg)
				if err != nil {
					fmt.Fprintf(os.Stderr,
						"Parse failed %s!:%v",
						getopt.OptArg, err)
					usage()
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Too many config!")
				usage()
				os.Exit(1)
			}
			cnt++
			mode = ModeDeleteLink

		case 'n':
			if cnt == 0 {
				veth1, err = parseNOption(getopt.OptArg)
				if err != nil {
					fmt.Fprintf(os.Stderr,
						"Parse failed %s!:%v",
						getopt.OptArg, err)
					usage()
					os.Exit(1)
				}
			} else if cnt == 1 {
				veth2, err = parseNOption(getopt.OptArg)
				if err != nil {
					fmt.Fprintf(os.Stderr,
						"Parse failed %s!:%v",
						getopt.OptArg, err)
					usage()
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Too many config!")
				usage()
				os.Exit(1)
			}
			cnt++

		case 'N':
			if cnt == 0 {
				veth1, err = parseNOption(getopt.OptArg)
				if err != nil {
					fmt.Fprintf(os.Stderr,
						"Parse failed %s!:%v",
						getopt.OptArg, err)
					usage()
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Too many config!")
				usage()
				os.Exit(1)
			}
			cnt++
			mode = ModeDeleteLink

		case 'x':
			vxlan, err = parseXOption(getopt.OptArg)
			mode = ModeAddVxlan

		case 'v':
			fmt.Printf("koko version: %s\n", VERSION)
			os.Exit(0)

		case 'h':
			usage()
			os.Exit(0)

		}

	}

	// Assuming everything else above has worked out -- we'll continue on and make the vth pair.
	// You'll node at this point we've created vEth data objects and pass them along to the makeVeth method.
	if mode != ModeAddVxlan && cnt == 2 {
		// case 1: two container endpoint.
		fmt.Printf("Create veth...")
		makeVeth(veth1, veth2)
		fmt.Printf("done\n")
	} else if mode == ModeAddVxlan && cnt == 1 {
		// case 2: one endpoint with vxlan
		fmt.Printf("Create vxlan %s\n", veth1.linkName)
		makeVxLan(veth1, vxlan)
	} else if mode == ModeDeleteLink && cnt == 1 {
		fmt.Printf("Delete link %s\n", veth1.linkName)
		veth1.removeVethLink()
	}

}
