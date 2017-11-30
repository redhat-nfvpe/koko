#!/usr/bin/env bats
# Acceptance test script for each command line options with IPv4/v6.
# https://github.com/redhat-nfvpe/koko/wiki/Examples

setup() {
# Create dockers and namespaces
    DOCKER1="$(docker run --rm -d -it --sysctl net.ipv6.conf.all.disable_ipv6=0 alpine sh)"
    DOCKER2="$(docker run --rm -d -it --sysctl net.ipv6.conf.all.disable_ipv6=0 alpine sh)"
    sudo ip netns add NS1
    sudo ip netns add NS2
    sudo ip netns exec NS1 ip link set up dev lo
    sudo ip netns exec NS2 ip link set up dev lo

    # vxNS1,vxNS2 is namespaces for vxlan (VNID) scope
    # In linux kernel, VNI should be unique in net namespace, hence
    # to use vxlan under veth, each veth pair should be different
    # namespace.
    sudo ip netns add vxNS1
    sudo ip netns add vxNS2
    sudo ./koko -n vxNS1,vxveth1,10.0.0.1/24 -n vxNS2,vxveth2,10.0.0.2/24
}

teardown() {
# Delete dockers and namespaces
    docker rm -f $DOCKER1
    docker rm -f $DOCKER2
    sudo ip netns del NS1
    sudo ip netns del NS2

    sudo ./koko -N vxNS1,vxveth1
    sudo ip netns del vxNS1
    sudo ip netns del vxNS2
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
      sudo ./koko -d $DOCKER1,vethD1D2,10.10.10.1/29,2001::1/64 \
                  -n NS2,vethNS2NS1,10.10.10.2/29,2001::2/64
      run docker exec $DOCKER1 ping -c 3 -w 5 10.10.10.2
      [ "$status" -eq 0 ]
      run docker exec $DOCKER1 ping6 -c 3 -w 5 2001::2
      [ "$status" -eq 0 ]
}

@test "Docker .. Docker (VXLAN)" {
      sudo ip netns exec vxNS1 ./koko -d $DOCKER1,vxlanD1D2,10.10.10.1/29,2001::1/64 -x vxveth1,10.0.0.2,100
      sudo ip netns exec vxNS2 ./koko -d $DOCKER2,vxlanD2D1,10.10.10.2/29,2001::2/64 -x vxveth2,10.0.0.1,100
      run docker exec $DOCKER1 ping -c 3 -w 5 10.10.10.2
      [ "$status" -eq 0 ]
      run docker exec $DOCKER1 ping6 -c 3 -w 5 2001::2
      [ "$status" -eq 0 ]
}


@test "netns .. netns (VXLAN)" {
      sudo ip netns exec vxNS1 ./koko -n NS1,vxlanNS1NS2,10.10.10.1/29,2001::1/64 -x vxveth1,10.0.0.2,100
      sudo ip netns exec vxNS2 ./koko -n NS2,vxlanNS2NS1,10.10.10.2/29,2001::2/64 -x vxveth2,10.0.0.1,100
      run sudo ip netns exec NS1 ping -c 3 -w 5 10.10.10.2
      [ "$status" -eq 0 ]
      run sudo ip netns exec NS1 ping6 -c 3 -w 5 2001::2
      [ "$status" -eq 0 ]
}

@test "Docker .. netns (VXLAN)" {
      sudo ip netns exec vxNS1 ./koko -d $DOCKER1,vxlanD1D2,10.10.10.1/29,2001::1/64 -x vxveth1,10.0.0.2,100
      sudo ip netns exec vxNS2 ./koko -n NS2,vxlanNS2NS1,10.10.10.2/29,2001::2/64 -x vxveth2,10.0.0.1,100
      run docker exec $DOCKER1 ping -c 3 -w 5 10.10.10.2
      [ "$status" -eq 0 ]
      run docker exec $DOCKER1 ping6 -c 3 -w 5 2001::2
      [ "$status" -eq 0 ]
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
