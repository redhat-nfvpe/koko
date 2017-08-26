/*
koko: Container connector
*/
package main

import (
	"fmt"
	"github.com/mattn/go-getopt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/MakeNowJust/heredoc"
	"github.com/redhat-nfvpe/koko/api"
)

// VERSION indicates koko's version.
var VERSION = "master@git"

// parseLinkIPOption parses '<linkname>(:<ip>/<prefix>)' syntax and put it in
// veth object
func parseLinkIPOption(veth *api.VEth, n []string) (err error) {
	veth.LinkName = n[0]
	numAddr := len(n) - 1

	exists,_ := api.IsExistLinkInNS(veth.NsName, veth.LinkName)
	if exists == true {
		return fmt.Errorf("exists interface %s at namespace (%s)",
				  veth.LinkName, veth.NsName)
	}

	veth.IPAddr = make([]net.IPNet, 0, numAddr)
	for i := 0; i < numAddr; i++ {
		// check mirror
		if (len(n[i+1]) > len("mirror:")) && (n[i+1][0:6] == "mirror") {
			n1 := strings.Split(n[i+1], ":")
			if len(n1) == 3 {
				switch n1[1] {
				case "ingress":
					veth.MirrorIngress = n1[2]
				case "egress":
					veth.MirrorEgress = n1[2]
				case "both":
					veth.MirrorEgress = n1[2]
					veth.MirrorIngress = n1[2]
				}
			} else {
				return fmt.Errorf("unknown mirror command: %s", n[i+1])
			}
		} else { // check CIDR (ip/prefixlen)
			ip, mask, err1 := net.ParseCIDR(n[i+1])
			if err1 != nil {
				return fmt.Errorf("failed to parse IP addr(%d) %s: %v",
				i, n[i], err1)
			}
			i := net.IPNet{
				IP: ip,
				Mask: mask.Mask,
			}
			veth.IPAddr = append(veth.IPAddr, i)
			// veth.IPAddr[i].IP = ip
			// veth.IPAddr[i].Mask = mask.Mask
		}
	}
	return
}

// parseNOption parses '-n' option and put this information in veth object.
func parseNOption(s string) (veth api.VEth, err error) {
	n := strings.Split(s, ",")
	if len(n) > 4 || len(n) < 1 {
		err = fmt.Errorf("failed to parse %s", s)
		return
	}

	veth.NsName = fmt.Sprintf("/var/run/netns/%s", n[0])
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

// parseCOption Parses '-c' option and put this information in veth object.
func parseCOption(s string) (veth api.VEth, err error) {
	n := strings.Split(s, ",")
	if len(n) > 4 || len(n) < 1 {
		err = fmt.Errorf("failed to parse %s", s)
		return
	}

	veth.NsName = ""

	err1 := parseLinkIPOption(&veth, n)
	if err1 != nil {
		fmt.Fprintf(os.Stderr, "%v", err1)
		os.Exit(1)
	}

	return
}

// parseDOption Parses '-d' option and put this information in veth object.
func parseDOption(s string) (veth api.VEth, err error) {
	n := strings.Split(s, ",")
	if len(n) > 4 || len(n) < 1 {
		err = fmt.Errorf("failed to parse %s", s)
		return
	}

	veth.NsName, err = api.GetDockerContainerNS(n[0])
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

// parsePOption Parses '-p' option and put this information in veth object.
func parsePOption(s string) (veth api.VEth, err error) {
	n := strings.Split(s, ",")
	if len(n) > 4 || len(n) < 1 {
		err = fmt.Errorf("failed to parse %s", s)
		return
	}

	veth.NsName = fmt.Sprintf("/proc/%s/ns/net", n[0])
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

// parseMOption parses '-M' option and put this information in veth object.
func parseMOption(s string) (macvlan api.MacVLan, err error) {

	n := strings.Split(s, ",")
	if len(n) != 2 {
		err = fmt.Errorf("failed to parse %s", s)
		return
	}

	macvlan.ParentIF = n[0]
	macvlanmode := strings.ToLower(n[1])

	switch macvlanmode {
	case "default":
		macvlan.Mode = netlink.MACVLAN_MODE_DEFAULT
	case "private":
		macvlan.Mode = netlink.MACVLAN_MODE_PRIVATE
	case "vepa":
		macvlan.Mode = netlink.MACVLAN_MODE_VEPA
	case "bridge":
		macvlan.Mode = netlink.MACVLAN_MODE_BRIDGE
	case "passthru":
		macvlan.Mode = netlink.MACVLAN_MODE_PASSTHRU
	}
	return
}

// parseVOption parses '-v' option and put this information in veth object.
func parseVOption(s string) (vlan api.VLan, err error) {
	var err2 error // if we encounter an error, it's marked here.

	n := strings.Split(s, ",")
	if len(n) != 2 {
		err = fmt.Errorf("failed to parse %s", s)
		return
	}

	vlan.ParentIF = n[0]
	vlan.ID, err2 = strconv.Atoi(n[1])
	if err2 != nil {
		err = fmt.Errorf("failed to parse VID %s: %v", n[1], err2)
		return
	}

	return
}

// parseXOption parses '-x' option and put this information in veth object.
func parseXOption(s string) (vxlan api.VxLan, err error) {
	var err2 error // if we encounter an error, it's marked here.

	n := strings.Split(s, ",")
	if len(n) != 3 {
		err = fmt.Errorf("failed to parse %s", s)
		return
	}

	vxlan.ParentIF = n[0]
	vxlan.IPAddr = net.ParseIP(n[1])
	vxlan.ID, err2 = strconv.Atoi(n[2])
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
		./koko -d centos1,link1 -c link2
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

* case8: connect docker/linux ns container to vxlan interface
./koko -d centos1:link1:192.168.1.1/24 -c link2

* case9: connect container of <pid1> and the one of <pid2>
./koko -p <pid1>:link1:192.168.1.1/24 -p <pid2>:link1:192.168.1.1/24

* case10: connect container of <pid1>
./koko -P <pid1>:link1

*/
func main() {

	var c int     // command line parameters.
	var err error // if we encounter an error, it's marked here.
	const (
		ModeUnspec = iota
		ModeAddVeth
		ModeAddVlan
		ModeAddVxlan
		ModeAddMacVlan
		ModeDeleteLink
	)

	cnt := 0 // Count of command line parameters.
	// Any errors with peeling apart the command line options.
	getopt.OptErr = 0

	// Create some empty vEth data objects.
	veth1 := api.VEth{}
	veth2 := api.VEth{}
	vxlan := api.VxLan{}
	vlan := api.VLan{}
	macvlan := api.MacVLan{}
	mode := ModeUnspec

	// Parse options and and exit if they don't meet our criteria.
	for {
		if c = getopt.Getopt("c:d:D:n:N:x:p:P:hvM:V:"); c == getopt.EOF {
			break
		}
		switch c {
		case 'd', 'D': // docker
			if cnt == 0 {
				veth1, err = parseDOption(getopt.OptArg)
				if err != nil {
					fmt.Fprintf(os.Stderr,
						"Parse failed %s!:%v",
						getopt.OptArg, err)
					usage()
					os.Exit(1)
				}
			} else if cnt == 1 && c == 'd' {
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
			if c == 'D' {
				mode = ModeDeleteLink
			}

		case 'p', 'P': // pid
			if cnt == 0 {
				veth1, err = parsePOption(getopt.OptArg)
				if err != nil {
					fmt.Fprintf(os.Stderr,
						"Parse failed %s!:%v",
						getopt.OptArg, err)
					usage()
					os.Exit(1)
				}
			} else if cnt == 1 && c == 'p' {
				veth2, err = parsePOption(getopt.OptArg)
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
			if c == 'P' {
				mode = ModeDeleteLink
			}

		case 'n', 'N': // linux netns
			if cnt == 0 {
				veth1, err = parseNOption(getopt.OptArg)
				if err != nil {
					fmt.Fprintf(os.Stderr,
						"Parse failed %s!:%v",
						getopt.OptArg, err)
					usage()
					os.Exit(1)
				}
			} else if cnt == 1 && c == 'n' {
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
			if c == 'N' {
				mode = ModeDeleteLink
			}

		case 'c': // current netns
			if cnt == 0 {
				veth1, err = parseCOption(getopt.OptArg)
				if err != nil {
					fmt.Fprintf(os.Stderr,
						"Parse failed %s!:%v",
						getopt.OptArg, err)
					usage()
					os.Exit(1)
				}
			} else if cnt == 1 {
				veth2, err = parseCOption(getopt.OptArg)
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

		case 'M': // MACVLAN
			macvlan, err = parseMOption(getopt.OptArg)
			mode = ModeAddMacVlan
			if err != nil {
				fmt.Fprintf(os.Stderr,
					    "Parse failed %s!:%v",
					    getopt.OptArg, err)
				usage()
				os.Exit(1)
			}

		case 'x', 'X': // VXLAN
			vxlan, err = parseXOption(getopt.OptArg)
			mode = ModeAddVxlan
			if err != nil {
				fmt.Fprintf(os.Stderr,
					    "Parse failed %s!:%v",
					    getopt.OptArg, err)
				usage()
				os.Exit(1)
			}

		case 'V': // VLAN
			vlan, err = parseVOption(getopt.OptArg)
			mode = ModeAddVlan
			if err != nil {
				fmt.Fprintf(os.Stderr,
					    "Parse failed %s!:%v",
					    getopt.OptArg, err)
				usage()
				os.Exit(1)
			}

		case 'v': // version
			fmt.Printf("koko version: %s\n", VERSION)
			os.Exit(0)

		case 'h': // help
			usage()
			os.Exit(0)

		}

	}

	// Assuming everything else above has worked out -- we'll continue
	// on and make the vth pair.
	// You'll node at this point we've created vEth data objects and
	// pass them along to the makeVeth method.
	if mode != ModeAddVxlan && cnt == 2 {
		// case 1: two container endpoint.
		fmt.Printf("Create veth...")
		err := api.MakeVeth(veth1, veth2)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nveth add failed: %v\n", err)
		} else {
			fmt.Printf("done\n")
		}
	} else if mode == ModeAddVxlan && cnt == 1 {
		// case 2: one endpoint with vxlan
		fmt.Printf("Create vxlan %s\n", veth1.LinkName)
		api.MakeVxLan(veth1, vxlan)
	} else if mode == ModeAddVlan && cnt == 1 {
		// case 3: one endpoint with vlan
		fmt.Printf("Create vlan %s\n", veth1.LinkName)
		api.MakeVLan(veth1, vlan)
	} else if mode == ModeAddMacVlan && cnt == 1 {
		// case 4: one endpoint with vlan
		fmt.Printf("Create macvlan %s\n", veth1.LinkName)
		api.MakeMacVLan(veth1, macvlan)
	} else if mode == ModeDeleteLink && cnt == 1 {
		fmt.Printf("Delete link %s\n", veth1.LinkName)
		if err := veth1.RemoveVethLink(); err != nil {
			fmt.Fprintf(os.Stderr, "\nveth delete failed: %v\n", err)
		}
	}

}
