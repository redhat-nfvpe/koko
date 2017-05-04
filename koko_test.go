package main

import (
	"bytes"
	"net"
	"strings"
	"testing"
)

func TestParseLinkIPOption(t *testing.T) {
	// test case1: parse "-n testns,testlink"
	str1 := "testlink"
	linkName1 := "testlink"
	veth1 := vEth{}

	err1 := parseLinkIPOption(&veth1, strings.Split(str1, ","))
	if err1 != nil {
		t.Fatalf("Parse error: %v", err1)
	}
	if veth1.linkName != linkName1 {
		t.Fatalf("nsName Parse error %s should be %s",
			veth1.linkName, linkName1)
	}

	// test case2: parse "testlink,192.168.1.1/24"
	str2 := "testlink,192.168.1.1/24"
	linkName2 := "testlink"
	linkIP4Addr2 := net.ParseIP("192.168.1.1")
	linkIP4Prefix2 := net.CIDRMask(24, 32)
	veth2 := vEth{}

	err2 := parseLinkIPOption(&veth2, strings.Split(str2, ","))
	if err2 != nil {
		t.Fatalf("Parse error: %v", err1)
	}
	if veth2.linkName != linkName2 {
		t.Fatalf("nsName Parse error %s should be %s",
			veth2.nsName, linkName2)
	}
	if !veth2.withIP4Addr {
		t.Fatalf("withIP4Addr should be true but %v",
			veth2.withIP4Addr)
	}
	if !veth2.ip4Addr.IP.Equal(linkIP4Addr2) {
		t.Fatalf("ip4Addr.IP Parse error %s should be %s",
			veth2.ip4Addr.IP, linkIP4Addr2)
	}
	if bytes.Compare(veth2.ip4Addr.Mask, linkIP4Prefix2) != 0 {
		t.Fatalf("ip4Addr.Mask Parse error %s should be %s",
			veth2.ip4Addr.Mask, linkIP4Prefix2)
	}
	if veth2.withIP6Addr {
		t.Fatal("withIP6Addr should be false")
	}

	// test case3: parse "testlink,192.168.1.1/24,ff02::1/64"
	str3 := "testlink,192.168.1.1/24,ff02::1/64"
	linkName3 := "testlink"
	linkIP4Addr3 := net.ParseIP("192.168.1.1")
	linkIP4Prefix3 := net.CIDRMask(24, 32)
	linkIP6Addr3 := net.ParseIP("ff02::1")
	linkIP6Prefix3 := net.CIDRMask(64, 128)
	veth3 := vEth{}

	err3 := parseLinkIPOption(&veth3, strings.Split(str3, ","))
	if err3 != nil {
		t.Fatalf("Parse error: %v", err1)
	}
	if veth3.linkName != linkName3 {
		t.Fatalf("nsName Parse error %s should be %s",
			veth3.nsName, linkName3)
	}
	if !veth3.withIP4Addr {
		t.Fatalf("withIP4Addr should be true")
	}
	if !veth3.ip4Addr.IP.Equal(linkIP4Addr3) {
		t.Fatalf("ip4Addr.IP Parse error %s should be %s",
			veth3.ip4Addr.IP, linkIP4Addr3)
	}
	if bytes.Compare(veth3.ip4Addr.Mask, linkIP4Prefix3) != 0 {
		t.Fatalf("ip4Addr.Mask Parse error %s should be %s",
			veth3.ip4Addr.Mask, linkIP4Prefix3)
	}
	if !veth3.withIP6Addr {
		t.Fatal("withIP6Addr should be true")
	}
	if !veth3.ip6Addr.IP.Equal(linkIP6Addr3) {
		t.Fatalf("ip4Addr.IP Parse error %s should be %s",
			veth3.ip6Addr.IP, linkIP6Addr3)
	}
	if bytes.Compare(veth3.ip6Addr.Mask, linkIP6Prefix3) != 0 {
		t.Fatalf("ip4Addr.Mask Parse error %s should be %s",
			veth3.ip6Addr.Mask, linkIP6Prefix3)
	}

	// test case4: parse "testlink,,ff02::1/64"
	str4 := "testlink,,ff02::1/64"
	linkName4 := "testlink"
	linkIP6Addr4 := net.ParseIP("ff02::1")
	linkIP6Prefix4 := net.CIDRMask(64, 128)
	veth4 := vEth{}

	err4 := parseLinkIPOption(&veth4, strings.Split(str4, ","))
	if err4 != nil {
		t.Fatalf("Parse error: %v", err1)
	}
	if veth4.linkName != linkName4 {
		t.Fatalf("nsName Parse error %s should be %s",
			veth4.nsName, linkName4)
	}
	if veth4.withIP4Addr {
		t.Fatal("withIP4Addr should be false")
	}
	if !veth4.withIP6Addr {
		t.Fatal("withIP6Addr should be true")
	}
	if !veth4.ip6Addr.IP.Equal(linkIP6Addr4) {
		t.Fatalf("ip4Addr.IP Parse error %s should be %s",
			veth4.ip6Addr.IP, linkIP6Addr4)
	}
	if bytes.Compare(veth4.ip6Addr.Mask, linkIP6Prefix4) != 0 {
		t.Fatalf("ip4Addr.Mask Parse error %s should be %s",
			veth4.ip6Addr.Mask, linkIP6Prefix4)
	}

}
