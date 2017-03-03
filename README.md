# vethcon: veth devices for containers (for chaining)

# Build
    go get
	go build

# Usage

    ./vethcon -d <container>:<linkname>:<ipaddr/mask> -d <container>:<linkname>:<ipaddr/mask>
    <container>: container identifier (CONTAINER ID or name)
	<linkname>: veth link name
	<ipaddr/mask>: IPv4 address with netmask (e.g. 192.168.1.1/24)
	Example:
	sudo ./vethcon -d centos1:link1:192.168.1.1/24 -d centos2:link2:192.168.1.2/24
