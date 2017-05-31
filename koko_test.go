package main

import (
	"bytes"
	"net"
	"strings"
	"testing"

	"github.com/redhat-nfvpe/koko/api"
)

func TestParseLinkIPOption(t *testing.T) {
	// test case1: parse "-n testns,testlink"
	str1 := "testlink"
	linkName1 := "testlink"
	veth1 := api.VEth{}

	err1 := parseLinkIPOption(&veth1, strings.Split(str1, ","))
	if err1 != nil {
		t.Fatalf("Parse error: %v", err1)
	}
	if veth1.LinkName != linkName1 {
		t.Fatalf("nsName Parse error %s should be %s",
			veth1.LinkName, linkName1)
	}

	// test case2: parse "testlink,192.168.1.1/24"
	str2 := "testlink,192.168.1.1/24"
	linkName2 := "testlink"
	linkIP4Addr2 := net.ParseIP("192.168.1.1")
	linkIP4Prefix2 := net.CIDRMask(24, 32)
	veth2 := api.VEth{}

	err2 := parseLinkIPOption(&veth2, strings.Split(str2, ","))
	if err2 != nil {
		t.Fatalf("Parse error: %v", err1)
	}
	if veth2.LinkName != linkName2 {
		t.Fatalf("nsName Parse error %s should be %s",
			veth2.NsName, linkName2)
	}
	if !veth2.IpAddr[0].IP.Equal(linkIP4Addr2) {
		t.Fatalf("ipAddr[0].IP Parse error %s should be %s",
			veth2.IpAddr[0].IP, linkIP4Addr2)
	}
	if bytes.Compare(veth2.IpAddr[0].Mask, linkIP4Prefix2) != 0 {
		t.Fatalf("ipAddr[0].Mask Parse error %s should be %s",
			veth2.IpAddr[0].Mask, linkIP4Prefix2)
	}

	// test case3: parse "testlink,192.168.1.1/24,ff02::1/64"
	str3 := "testlink,192.168.1.1/24,ff02::1/64"
	linkName3 := "testlink"
	linkIP4Addr3 := net.ParseIP("192.168.1.1")
	linkIP4Prefix3 := net.CIDRMask(24, 32)
	linkIP6Addr3 := net.ParseIP("ff02::1")
	linkIP6Prefix3 := net.CIDRMask(64, 128)
	veth3 := api.VEth{}

	err3 := parseLinkIPOption(&veth3, strings.Split(str3, ","))
	if err3 != nil {
		t.Fatalf("Parse error: %v", err1)
	}
	if veth3.LinkName != linkName3 {
		t.Fatalf("nsName Parse error %s should be %s",
			veth3.NsName, linkName3)
	}
	if !veth3.IpAddr[0].IP.Equal(linkIP4Addr3) {
		t.Fatalf("ipAddr[0].IP Parse error %s should be %s",
			veth3.IpAddr[0].IP, linkIP4Addr3)
	}
	if bytes.Compare(veth3.IpAddr[0].Mask, linkIP4Prefix3) != 0 {
		t.Fatalf("ipAddr[0].Mask Parse error %s should be %s",
			veth3.IpAddr[0].Mask, linkIP4Prefix3)
	}
	if !veth3.IpAddr[1].IP.Equal(linkIP6Addr3) {
		t.Fatalf("ipAddr[1].IP Parse error %s should be %s",
			veth3.IpAddr[1].IP, linkIP6Addr3)
	}
	if bytes.Compare(veth3.IpAddr[1].Mask, linkIP6Prefix3) != 0 {
		t.Fatalf("ipAddr[1].Mask Parse error %s should be %s",
			veth3.IpAddr[1].Mask, linkIP6Prefix3)
	}

	// test case4: parse "testlink,ff02::1/64"
	str4 := "testlink,ff02::1/64"
	linkName4 := "testlink"
	linkIP6Addr4 := net.ParseIP("ff02::1")
	linkIP6Prefix4 := net.CIDRMask(64, 128)
	veth4 := api.VEth{}

	err4 := parseLinkIPOption(&veth4, strings.Split(str4, ","))
	if err4 != nil {
		t.Fatalf("Parse error: %v", err1)
	}
	if veth4.LinkName != linkName4 {
		t.Fatalf("nsName Parse error %s should be %s",
			veth4.NsName, linkName4)
	}
	if !veth4.IpAddr[0].IP.Equal(linkIP6Addr4) {
		t.Fatalf("ipAddr[0].IP Parse error %s should be %s",
			veth4.IpAddr[0].IP, linkIP6Addr4)
	}
	if bytes.Compare(veth4.IpAddr[0].Mask, linkIP6Prefix4) != 0 {
		t.Fatalf("ipAddr[0].Mask Parse error %s should be %s",
			veth4.IpAddr[0].Mask, linkIP6Prefix4)
	}

}
