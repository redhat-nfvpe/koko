# koko: Container connector (for chaining)

[![Build Status](https://travis-ci.org/redhat-nfvpe/koko.svg?branch=master)](https://travis-ci.org/redhat-nfvpe/koko)

# What is 'koko'?

`koko` is a simple tool which connects between Docker containers/linux netns processes with veth devices/vxlan
of linux kernel. Using `koko`, you can simply make point-to-point connection for containers without linux bridges.

# Build

`koko` is written in go, so following commands makes `koko` single binary. Build and put it in your container host.

    go getdevice
    go build

# Syntax

`koko` takes two arguments: two endpoints of container and `koko` connects both.
`koko` supports veth for two containers in one host and vxlan for two containers in separate host.

## Connecting containers in container host using veth

    ./koko [-d <container>:<linkname> |
            -d <container>:<linkname>:<IPv4 addr>/<prefixlen> |
            -n <netns name>:<linkname> |
            -n <netns name>:<linkname>:<IPv4 addr>/<prefixlen> ]
           [-d <container>:<linkname> |
            -d <container>:<linkname>:<IPv4 addr>/<prefixlen> |
            -n <netns name>:<linkname> |
            -n <netns name>:<linkname>:<IPv4 addr>/<prefixlen> ]

## Connecting containers using vxlan (interconnecting container hosts)

Connecting containers which are in separate hosts with vxlan. Following command makes vxlan interface 
and put this interface into given container with/without IP address.

    ./koko [-d <container>:<linkname> |
            -d <container>:<linkname>:<IPv4 addr>/<prefixlen> |
            -n <netns name>:<linkname> |
            -n <netns name>:<linkname>:<IPv4 addr>/<prefixlen> ]
           [-x <parent interface>:<remote endpoint IP addr>:<vxlan id> ]

## Printing help

    ./koko -h

# Usage

    # Print usage instruction
    ./koko -h

    # Config veth without IPv4 addr for Docker container
    ./koko -d <container>:<linkname> -d <container>:<linkname>
    <container>: Docker's container identifier (CONTAINER ID or name)
    <linkname>: veth link name

    # Config veth with IPv4 addr for Docker container
    ./koko -d <container>:<linkname>:<ipaddr/mask> -d <container>:<linkname>:<ipaddr>/<prefixlen>
    <container>: Docker's container identifier (CONTAINER ID or name)
    <linkname>: veth link name
    <ipaddr>/<prefixlen>: IPv4 address with netmask (e.g. 192.168.1.1/24)

    # Config vxlan with IPv4 addr for Docker container
    ./koko -d <container>:<linkname>:<ipaddr/mask> -x <parent IF>:<remote IP>:<vxlan id>
    <container>: Docker's container identifier (CONTAINER ID or name)
    <linkname>: veth link name
    <ipaddr>/<prefixlen>: IPv4 address with netmask (e.g. 192.168.1.1/24)
    <parent IF>: Egress IF name for vxlan (e.g. eth0)
    <remote IP>: Unicast destination IP address for endpoint
    <vxlan id>: vxlan id

    # Config veth without IPv4 addr for netns
    ./koko -n <netns name>:<linkname> -n <netns name>:<linkname>
    <netns name>: netns name that is given by 'ip netns' command
    <linkname>: veth link name
    <ipaddr>/<prefixlen>: IPv4 address with netmask (e.g. 192.168.1.1/24)

    # Config veth with IPv4 addr for netns
    ./koko -n <netns name>:<linkname>:<ipaddr/mask> -n <netns name>:<linkname>:<ipaddr>/<prefixlen>
    <netns name>: netns name that is given by 'ip netns' command
    <linkname>: veth link name
    <ipaddr>/<prefixlen>: IPv4 address with netmask (e.g. 192.168.1.1/24)

    # Config vxlan with IPv4 addr for netns
    ./koko -n <netns name>:<linkname>:<ipaddr/mask> -x <parent IF>:<remote IP>:<vxlan id>
    <netns name>: netns name that is given by 'ip netns' command
    <linkname>: veth link name
    <ipaddr>/<prefixlen>: IPv4 address with netmask (e.g. 192.168.1.1/24)
    <parent IF>: Egress IF name for vxlan (e.g. eth0)
    <remote IP>: Unicast destination IP address for endpoint
    <vxlan id>: vxlan id

## Example

    # connect between docker containers
    sudo ./koko -d centos1:link1:192.168.1.1/24 -d centos2:link2:192.168.1.2/24
    # connect between netns namespaces
    sudo ./koko -n testns1:link1:192.168.1.1/24 -n testns2:link2:192.168.1.2/24
    # connect between docker container and netns namespace
    sudo ./koko -d centos1:link1:192.168.1.1/24 -n testns2:link2:192.168.1.2/24
    # create vxlan interface and put it into docker container
    sudo ./koko -d centos1:link1:192.168.1.1/24 -x eth1:10.1.1.1:1

# Todo
- Document
- Logo?

# Authors
- Tomofumi Hayashi (s1061123)
- Doug Smith (dougbtv)
