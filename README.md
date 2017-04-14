# koko: Container connector (for chaining)

[![Build Status](https://travis-ci.org/redhat-nfvpe/koko.svg?branch=master)](https://travis-ci.org/redhat-nfvpe/koko)

# What is 'koko'?

`koko` is a simple tool which connects between Docker containers/linux netns processes with veth devices/vxlan
of linux kernel. Using `koko`, you can simply make point-to-point connection for containers without linux bridges.

![Koko's abstruct design](https://raw.githubusercontent.com/wiki/redhat-nfvpe/koko/images/koko.png)

# Build

`koko` is written in go, so following commands makes `koko` single binary. Build and put it in your container host.

    go getdevice
    go build

# Syntax

`koko` takes two arguments: two endpoints of container and `koko` connects both.
`koko` supports veth for two containers in one host and vxlan for two containers in separate host.

## Connecting containers in container host using veth

    ./koko {-d <container>:<linkname>[:<IPv4 addr>/<prefixlen>] |
            -n <netns name>:<linkname>[:<IPv4 addr>/<prefixlen>] }
           {-d <container>:<linkname>[:<IPv4 addr>/<prefixlen>] |
            -n <netns name>:<linkname>[:<IPv4 addr>/<prefixlen>] }

## Connecting containers using vxlan (interconnecting container hosts)

Connecting containers which are in separate hosts with vxlan. Following command makes vxlan interface 
and put this interface into given container with/without IP address.

    ./koko {-d <container>:<linkname>[:<IPv4 addr>/<prefixlen>] |
            -n <netns name>:<linkname>[:<IPv4 addr>/<prefixlen>] }
            -x <parent interface>:<remote endpoint IP addr>:<vxlan id> 

## Printing help

    ./koko -h

# Usage
Please see [Examples](https://github.com/redhat-nfvpe/koko/wiki/Examples) in Wiki.

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
