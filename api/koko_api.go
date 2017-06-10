/*
Package api provides koko's connector funcitionlity as API.
*/
package api

import (
	"fmt"
	"net"
	"os"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"

	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

// VEth is a structure to descrive veth interfaces.
type VEth struct {
	NsName   string      // What's the network namespace?
	LinkName string      // And what will we call the link.
	IPAddr   []net.IPNet // Slice of IPv4/v6 address.
}

// VxLan is a structure to descrive vxlan endpoint.
type VxLan struct {
	ParentIF string // parent interface name
	ID       int    // VxLan ID
	IPAddr   net.IP // VxLan destination address
}

// MakeVethPair makes veth pair and returns its link.
func MakeVethPair(name, peer string, mtu int) (netlink.Link, error) {
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

// GetVethPair takes two link names and create a veth pair and return
//both links.
func GetVethPair(name1 string, name2 string) (link1 netlink.Link,
	link2 netlink.Link, err error) {
	link1, err = MakeVethPair(name1, name2, 1500)
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

// AddVxLanInterface creates VxLan interface by given vxlan object
func AddVxLanInterface(vxlan VxLan, devName string) error {
	parentIF, err := netlink.LinkByName(vxlan.ParentIF)

	if err != nil {
		return fmt.Errorf("Failed to get %s: %v", vxlan.ParentIF, err)
	}

	vxlanconf := netlink.Vxlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:   devName,
			TxQLen: 1000,
		},
		VxlanId:      vxlan.ID,
		VtepDevIndex: parentIF.Attrs().Index,
		Group:        vxlan.IPAddr,
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

// GetDockerContainerNS retrieves container's network namespace from
// docker container id, given as containerID.
func GetDockerContainerNS(containerID string) (namespace string, err error) {
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

// SetVethLink is low-level handler to set IP address onveth links given
// a single VEth data object.
// ...primarily used privately by makeVeth().
func (veth *VEth) SetVethLink(link netlink.Link) (err error) {
	var vethNs ns.NetNS

	if veth.NsName == "" {
		vethNs, err = ns.GetCurrentNS()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
		}
	} else {
		vethNs, err = ns.GetNS(veth.NsName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
		}
	}

	defer vethNs.Close()
	if err = netlink.LinkSetNsFd(link, int(vethNs.Fd())); err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}

	err = vethNs.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(veth.LinkName)
		if err != nil {
			return fmt.Errorf("failed to lookup %q in %q: %v",
				veth.LinkName, vethNs.Path(), err)
		}

		if err = netlink.LinkSetUp(link); err != nil {
			return fmt.Errorf("failed to set %q up: %v",
				veth.LinkName, err)
		}

		// Conditionally set the IP address.
		for i := 0; i < len(veth.IPAddr); i++ {
			addr := &netlink.Addr{IPNet: &veth.IPAddr[i], Label: ""}
			if err = netlink.AddrAdd(link, addr); err != nil {
				return fmt.Errorf(
					"failed to add IP addr %v to %q: %v",
					addr, veth.LinkName, err)
			}
		}

		return nil
	})

	return
}

// RemoveVethLink is low-level handler to get interface handle in
// container/netns namespace and remove it.
func (veth *VEth) RemoveVethLink() (err error) {
	var vethNs ns.NetNS

	if veth.NsName == "" {
		vethNs, err = ns.GetCurrentNS()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
		}
	} else {
		vethNs, err = ns.GetNS(veth.NsName)

		if err != nil {
			fmt.Fprintf(os.Stderr, "%v", err)
		}
	}
	defer vethNs.Close()

	err = vethNs.Do(func(_ ns.NetNS) error {
		link, err := netlink.LinkByName(veth.LinkName)
		if err != nil {
			return fmt.Errorf("failed to lookup %q in %q: %v",
				veth.LinkName, vethNs.Path(), err)
		}

		err = netlink.LinkDel(link)
		if err != nil {
			return fmt.Errorf("failed to remove link %q in %q: %v",
				veth.LinkName, vethNs.Path(), err)
		}
		return nil
	})

	return
}

// MakeVeth is top-level handler to create veth links given two VEth data
// objects: veth1 and veth2.
func MakeVeth(veth1 VEth, veth2 VEth) {

	link1, link2, err := GetVethPair(veth1.LinkName, veth2.LinkName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
	}

	veth1.SetVethLink(link1)
	veth2.SetVethLink(link2)
}

// MakeVxLan makes vxlan interface and put it into container namespace
func MakeVxLan(veth1 VEth, vxlan VxLan) {

	err := AddVxLanInterface(vxlan, veth1.LinkName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vxlan add failed: %v", err)
	}

	link, err2 := netlink.LinkByName(veth1.LinkName)
	if err2 != nil {
		fmt.Fprintf(os.Stderr, "Cannot get %s: %v", veth1.LinkName, err)
	}
	err = veth1.SetVethLink(link)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot add IPaddr/netns failed: %v",
			err)
	}
}
