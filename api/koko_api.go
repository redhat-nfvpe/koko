/*
Package api provides koko's connector funcitionlity as API.
*/
package api

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"

	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

// VEth is a structure to descrive veth interfaces.
type VEth struct {
	NsName   string      // What's the network namespace?
	LinkName string      // And what will we call the link.
	IPAddr   []net.IPNet // (optional) Slice of IPv4/v6 address.
	MirrorEgress string  // (optional) source interface for egress mirror
	MirrorIngress string // (optional) source interface for ingress mirror
}

// VxLan is a structure to descrive vxlan endpoint.
type VxLan struct {
	ParentIF string // parent interface name
	ID       int    // VxLan ID
	IPAddr   net.IP // VxLan destination address
}

// VLan is a structure to descrive vlan endpoint.
type VLan struct {
	ParentIF string // parent interface name
	ID       int    // VLan ID
}

// MacVLan is a structure to descrive vlan endpoint.
type MacVLan struct {
	ParentIF string			// parent interface name
	Mode	 netlink.MacvlanMode	// MacVlan mode
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

	if link2, err = netlink.LinkByName(name2); err != nil {
		err = fmt.Errorf("Failed to lookup %q: %v\n", name2, err)
	}

	return
}

// AddVxLanInterface creates VxLan interface by given vxlan object
func AddVxLanInterface(vxlan VxLan, devName string) (err error) {
	var parentIF netlink.Link

	if parentIF, err = netlink.LinkByName(vxlan.ParentIF); err != nil {
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

// AddVLanInterface creates VLan interface by given vlan object
func AddVLanInterface(vlan VLan, devName string) (err error) {
	var parentIF netlink.Link

	if parentIF, err = netlink.LinkByName(vlan.ParentIF); err != nil {
		return fmt.Errorf("Failed to get %s: %v", vlan.ParentIF, err)
	}

	vlanconf := netlink.Vlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:   devName,
			ParentIndex: parentIF.Attrs().Index,
		},
		VlanId: vlan.ID,
	}

	if err = netlink.LinkAdd(&vlanconf); err != nil {
		return fmt.Errorf("Failed to add vlan %s: %v", devName, err)
	}
	return nil
}

// AddMacVLanInterface creates MacVLan interface by given macvlan object
func AddMacVLanInterface(macvlan MacVLan, devName string) (err error) {
	var parentIF netlink.Link

	if parentIF, err = netlink.LinkByName(macvlan.ParentIF); err != nil {
		return fmt.Errorf("Failed to get %s: %v", macvlan.ParentIF, err)
	}

	macvlanconf := netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:   devName,
			ParentIndex: parentIF.Attrs().Index,
		},
		Mode: macvlan.Mode,
	}

	if err = netlink.LinkAdd(&macvlanconf); err != nil {
		return fmt.Errorf("Failed to add vlan %s: %v", devName, err)
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

	cli.NegotiateAPIVersion(ctx)

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

// SetIngressMirror sets TC to mirror ingress from given port
// as MirrorIngress.
func (veth *VEth) SetIngressMirror() (err error) {
	var linkSrc, linkDest netlink.Link

	if linkSrc, err = netlink.LinkByName(veth.MirrorIngress); err != nil {
		return fmt.Errorf("failed to lookup %q in %q: %v",
			veth.MirrorIngress, veth.NsName, err)
	}

	if linkDest, err = netlink.LinkByName(veth.LinkName); err != nil {
		return fmt.Errorf("failed to lookup %q in %q: %v",
			veth.LinkName, veth.NsName, err)
	}

	// tc qdisc add dev $SRC_IFACE ingress
	qdisc := &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: linkSrc.Attrs().Index,
			Handle: netlink.MakeHandle(0xffff, 0),
			Parent: netlink.HANDLE_INGRESS,
		},
	}
	if err = netlink.QdiscAdd(qdisc); err != nil {
		if ! os.IsExist(err) {
			return err
		}
	}

	// tc filter add dev $SRC_IFACE parent fffff:
	// protocol all
	// u32 match u32 0 0
	// action mirred egress mirror dev $DST_IFACE
	filter := &netlink.U32 {
		FilterAttrs: netlink.FilterAttrs {
			LinkIndex: linkSrc.Attrs().Index,
			Parent: netlink.MakeHandle(0xffff, 0),
			Protocol: syscall.ETH_P_ALL,
		},
		Sel: &netlink.TcU32Sel{
			Keys: []netlink.TcU32Key {
				netlink.TcU32Key{
					Mask: 0x0,
					Val: 0,
				},
			},
			Flags: netlink.TC_U32_TERMINAL,
		},
		Actions: []netlink.Action{
			&netlink.MirredAction{
				ActionAttrs: netlink.ActionAttrs{
					Action: netlink.TC_ACT_PIPE,
				},
				MirredAction: netlink.TCA_EGRESS_MIRROR,
				Ifindex: linkDest.Attrs().Index,
			},
		},
	}

	if err = netlink.FilterAdd(filter); err != nil {
		return err
	}
	return nil
}

// SetEgressMirror sets TC to mirror egress from given port
// as MirrorEgress.
func (veth *VEth) SetEgressMirror() (err error) {
	var linkSrc, linkDest netlink.Link

	if linkSrc, err = netlink.LinkByName(veth.MirrorEgress); err != nil {
		return fmt.Errorf("failed to lookup %q in %q: %v",
			veth.MirrorEgress, veth.NsName, err)
	}

	if linkSrc.Attrs().TxQLen == 0 {
		return fmt.Errorf("veth qlen must be non zero!")
	}

	if linkDest, err = netlink.LinkByName(veth.LinkName); err != nil {
		return fmt.Errorf("failed to lookup %q in %q: %v",
			veth.LinkName, veth.NsName, err)
	}

	// tc qdisc add dev <SRC> handle 1: root prio
	qdisc := netlink.NewPrio(
		netlink.QdiscAttrs{
			LinkIndex: linkSrc.Attrs().Index,
			Handle: netlink.MakeHandle(1, 0),
			Parent: netlink.HANDLE_ROOT,
		})
	if err = netlink.QdiscAdd(qdisc); err != nil {
		if ! os.IsExist(err) {
			return err
		}
	}
	// tc filter add dev $SRC_IFACE parent 1:
	// protocol all
	// u32 match u32 0 0
	// action mirred egress mirror dev $DST_IFACE
	u32SelKeys := []netlink.TcU32Key {
		netlink.TcU32Key{
			Mask: 0x0,
			Val: 0,
		},
	}
	filter := &netlink.U32 {
		FilterAttrs: netlink.FilterAttrs {
			LinkIndex: linkSrc.Attrs().Index,
			Parent: netlink.MakeHandle(1, 0),
			Protocol: syscall.ETH_P_ALL,
		},
		Sel: &netlink.TcU32Sel{
			Keys: u32SelKeys,
			Flags: netlink.TC_U32_TERMINAL,
		},
		Actions: []netlink.Action{
			&netlink.MirredAction{
				ActionAttrs: netlink.ActionAttrs{
					Action: netlink.TC_ACT_PIPE,
				},
				MirredAction: netlink.TCA_EGRESS_MIRROR,
				Ifindex: linkDest.Attrs().Index,
			},
		},
	}

	if err = netlink.FilterAdd(filter); err != nil {
		return err
	}
	return nil
}

// UnsetIngressMirror sets TC to mirror ingress from given port
// as MirrorIngress.
func (veth *VEth) UnsetIngressMirror() (err error) {
	var linkSrc netlink.Link

	if linkSrc, err = netlink.LinkByName(veth.MirrorIngress); err != nil {
		return fmt.Errorf("failed to lookup %q in %q: %v",
			veth.MirrorIngress, veth.NsName, err)
	}

	// tc qdisc add dev $SRC_IFACE ingress
	qdisc := &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: linkSrc.Attrs().Index,
			Handle: netlink.MakeHandle(0xffff, 0),
			Parent: netlink.HANDLE_INGRESS,
		},
	}
	if err = netlink.QdiscDel(qdisc); err != nil {
		return err
	}

	return nil
}

// UnsetEgressMirror sets TC to mirror egress from given port
// as MirrorEgress.
func (veth *VEth) UnsetEgressMirror() (err error) {
	var linkSrc netlink.Link

	if linkSrc, err = netlink.LinkByName(veth.MirrorEgress); err != nil {
		return fmt.Errorf("failed to lookup %q in %q: %v",
			veth.MirrorEgress, veth.NsName, err)
	}

	// tc qdisc add dev <SRC> handle 1: root prio
	qdisc := netlink.NewPrio(
		netlink.QdiscAttrs{
			LinkIndex: linkSrc.Attrs().Index,
			Handle: netlink.MakeHandle(1, 0),
			Parent: netlink.HANDLE_ROOT,
		})
	if err = netlink.QdiscDel(qdisc); err != nil {
		return err
	}
	return nil
}

// SetVethLink is low-level handler to set IP address onveth links given
// a single VEth data object.
// ...primarily used privately by makeVeth().
func (veth *VEth) SetVethLink(link netlink.Link) (err error) {
	var vethNs ns.NetNS

	if veth.NsName == "" {
		if vethNs, err = ns.GetCurrentNS(); err != nil {
			return fmt.Errorf("%v", err)
		}
	} else {
		if vethNs, err = ns.GetNS(veth.NsName); err != nil {
			return fmt.Errorf("%v", err)
		}
	}

	defer vethNs.Close()
	if err = netlink.LinkSetNsFd(link, int(vethNs.Fd())); err != nil {
		return fmt.Errorf("%v", err)
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

		if veth.MirrorIngress != "" {
			if err = veth.SetIngressMirror(); err != nil {
				netlink.LinkDel(link)
				return fmt.Errorf(
					"Failed to set tc ingress mirror :%v",
				err)
			}
		}
		if veth.MirrorEgress != "" {
			if err = veth.SetEgressMirror(); err != nil {
				netlink.LinkDel(link)
				return fmt.Errorf(
					"Failed to set tc egress mirror: %v", err)
			}
		}
		return nil
	})

	return err
}

// RemoveVethLink is low-level handler to get interface handle in
// container/netns namespace and remove it.
func (veth *VEth) RemoveVethLink() (err error) {
	var vethNs ns.NetNS
	var link netlink.Link

	if veth.NsName == "" {
		if vethNs, err = ns.GetCurrentNS(); err != nil {
			return fmt.Errorf("%v", err)
		}
	} else {
		if vethNs, err = ns.GetNS(veth.NsName); err != nil {
			return fmt.Errorf("%v", err)
		}
	}
	defer vethNs.Close()

	err = vethNs.Do(func(_ ns.NetNS) error {
		if veth.MirrorIngress != "" {
			if err = veth.UnsetIngressMirror(); err != nil {
				return fmt.Errorf(
					"Failed to unset tc ingress mirror :%v",
				err)
			}
		}
		if veth.MirrorEgress != "" {
			if err = veth.UnsetEgressMirror(); err != nil {
				return fmt.Errorf(
					"Failed to unset tc egress mirror: %v",
				err)
			}
		}

		if link, err = netlink.LinkByName(veth.LinkName); err != nil {
			return fmt.Errorf("failed to lookup %q in %q: %v",
				veth.LinkName, vethNs.Path(), err)
		}

		if err = netlink.LinkDel(link); err != nil {
			return fmt.Errorf("failed to remove link %q in %q: %v",
				veth.LinkName, vethNs.Path(), err)
		}
		return nil
	})

	return err
}

// MakeVeth is top-level handler to create veth links given two VEth data
// objects: veth1 and veth2.
func MakeVeth(veth1 VEth, veth2 VEth) (error) {
	link1, link2, err := GetVethPair(veth1.LinkName, veth2.LinkName)
	if err != nil {
		return err
	}

	if err:= veth1.SetVethLink(link1); err != nil {
		return err
	}
	if err:= veth2.SetVethLink(link2); err != nil {
		return err
	}
	return nil
}

// MakeVxLan makes vxlan interface and put it into container namespace
func MakeVxLan(veth1 VEth, vxlan VxLan) (err error) {
	var link netlink.Link

	if err = AddVxLanInterface(vxlan, veth1.LinkName); err != nil {
		return fmt.Errorf("vxlan add failed: %v", err)
	}

	if link, err = netlink.LinkByName(veth1.LinkName); err != nil {
		return fmt.Errorf("Cannot get %s: %v", veth1.LinkName, err)
	}

	if err = veth1.SetVethLink(link); err != nil {
		return fmt.Errorf("Cannot add IPaddr/netns failed: %v", err)
	}

	if veth1.MirrorIngress != "" {
		if err = veth1.SetIngressMirror(); err != nil {
			netlink.LinkDel(link)
			return fmt.Errorf(
				"Failed to set tc ingress mirror :%v",
				err)
		}
	}
	if veth1.MirrorEgress != "" {
		if err = veth1.SetEgressMirror(); err != nil {
			netlink.LinkDel(link)
			return fmt.Errorf(
				"Failed to set tc egress mirror: %v", err)
		}
	}
	return nil
}

// MakeVLan makes vlan interface
func MakeVLan(veth1 VEth, vlan VLan) (err error) {
	var link netlink.Link

	if err = AddVLanInterface(vlan, veth1.LinkName); err != nil {
		return fmt.Errorf("vlan add failed: %v", err)
	}

	if link, err = netlink.LinkByName(veth1.LinkName); err != nil {
		return fmt.Errorf("Cannot get %s: %v", veth1.LinkName, err)
	}
	if err = veth1.SetVethLink(link); err != nil {
		return fmt.Errorf("Cannot add IPaddr/netns failed: %v", err)
	}

	if veth1.MirrorIngress != "" {
		if err = veth1.SetIngressMirror(); err != nil {
			netlink.LinkDel(link)
			return fmt.Errorf(
				"Failed to set tc ingress mirror :%v",
				err)
		}
	}
	if veth1.MirrorEgress != "" {
		if err = veth1.SetEgressMirror(); err != nil {
			netlink.LinkDel(link)
			return fmt.Errorf(
				"Failed to set tc egress mirror: %v", err)
		}
	}
	return nil
}

// MakeMacVLan makes macvlan interface
func MakeMacVLan(veth1 VEth, macvlan MacVLan) (err error) {
	var link netlink.Link

	if err = AddMacVLanInterface(macvlan, veth1.LinkName); err != nil {
		return fmt.Errorf("macvlan add failed: %v", err)
	}

	if link, err = netlink.LinkByName(veth1.LinkName); err != nil {
		return fmt.Errorf("Cannot get %s: %v", veth1.LinkName, err)
	}

	if err = veth1.SetVethLink(link); err != nil {
		fmt.Errorf("Cannot add IPaddr/netns failed: %v", err)
	}
	if veth1.MirrorIngress != "" {
		if err = veth1.SetIngressMirror(); err != nil {
			netlink.LinkDel(link)
			return fmt.Errorf(
				"Failed to set tc ingress mirror :%v",
				err)
		}
	}
	if veth1.MirrorEgress != "" {
		if err = veth1.SetEgressMirror(); err != nil {
			netlink.LinkDel(link)
			return fmt.Errorf(
				"Failed to set tc egress mirror: %v", err)
		}
	}
	return err
}
