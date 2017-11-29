#!/usr/bin/env bats
# Acceptance test script for each command line options with IPv4/v6.
# https://github.com/redhat-nfvpe/koko/wiki/Examples

setup() {
# Create dockers and namespaces
    DOCKER1="$(docker run --rm -d -it --sysctl net.ipv6.conf.all.disable_ipv6=0 alpine sh)"
    DOCKER2="$(docker run --rm -d -it --sysctl net.ipv6.conf.all.disable_ipv6=0 alpine sh)"
    sudo ip netns add NS1
    sudo ip netns add NS2
}

teardown() {
# Delete dockers and namespaces
    docker rm -f $DOCKER1
    docker rm -f $DOCKER2
    sudo ip netns del NS1
    sudo ip netns del NS2
}



@test "Docker .. Docker" {
      sudo ./koko -d $DOCKER1,vethD1D2,10.10.10.1/29,2001::1/64 \
             -d $DOCKER2,vethD2D1,10.10.10.2/29,2001::2/64
      run docker exec $DOCKER1 ping -c 3 -w 5 10.10.10.2  
      [ "$status" -eq 0 ]
      run docker exec $DOCKER1 ping6 -c 3 -w 5 2001::2
      [ "$status" -eq 0 ]
}

@test "netns .. netns" {
      sudo ./koko -n NS1,vethNS1NS2,10.10.10.1/29,2001::1/64 \
                  -n NS2,vethNS2NS1,10.10.10.2/29,2001::2/64
      run sudo ip netns exec NS1 ping -c 3 -w 5 10.10.10.2  
      [ "$status" -eq 0 ]
      run sudo ip netns exec NS1 ping6 -c 3 -w 5 2001::2
      [ "$status" -eq 0 ]
}

@test "Docker .. netns" {
      skip
      echo "TODO"
}

@test "Docker .. Docker (VXLAN)" {
      skip
      echo "TODO"
}


@test "netns .. netns (VXLAN)" {
      skip
      echo "TODO"
}

@test "Docker .. netns (VXLAN)" {
      skip
      echo "TODO"
}

@test "Docker .. VLAN" {
      skip
      echo "TODO"
}

@test "netns .. VLAN" {
      skip
      echo "TODO"
}

@test "Docker .. macvlan" {
      skip
      echo "TODO"
}


@test "Docker .. Docker (Mirroring)" {
      skip
      echo "TODO"
}


@test "Docker .. VXLAN (Mirroring)" {
      skip
      echo "TODO"
}
