# vethcon: veth devices for containers (for chaining)

[![Build Status](https://travis-ci.org/s1061123/vethcon.svg?branch=master)](https://travis-ci.org/s1061123/vethcon)

# Build
    go get
    go build

# Syntax
`vethcon` takes two arguments: two endpoints of container and `vethcon` connects both.

    ./vethcon [-d <container>:<linkname> |
               -d <container>:<linkname>:<IPv4 addr>/<prefixlen> |
               -n <netns name>:<linkname> |
               -n <netns name>:<linkname>:<IPv4 addr>/<prefixlen> ]
              [-d <container>:<linkname> |
               -d <container>:<linkname>:<IPv4 addr>/<prefixlen> |
               -n <netns name>:<linkname> |
               -n <netns name>:<linkname>:<IPv4 addr>/<prefixlen> ]

# Usage

    # Config veth without IPv4 addr
    ./vethcon -d <container>:<linkname> -d <container>:<linkname>
    <container>: Docker's container identifier (CONTAINER ID or name)
    <linkname>: veth link name

    # Config veth with IPv4 addr
    ./vethcon -d <container>:<linkname>:<ipaddr/mask> -d <container>:<linkname>:<ipaddr>/<prefixlen>
    <container>: Docker's container identifier (CONTAINER ID or name)
    <linkname>: veth link name
    <ipaddr>/<prefixlen>: IPv4 address with netmask (e.g. 192.168.1.1/24)

## Example

    # connect between docker containers
    sudo ./vethcon -d centos1:link1:192.168.1.1/24 -d centos2:link2:192.168.1.2/24
    # connect between netns namespaces
    sudo ./vethcon -n testns1:link1:192.168.1.1/24 -n testns2:link2:192.168.1.2/24
    # connect between docker container and netns namespace
    sudo ./vethcon -d centos1:link1:192.168.1.1/24 -n testns2:link2:192.168.1.2/24

# Todo
- Add more good name
- Document
