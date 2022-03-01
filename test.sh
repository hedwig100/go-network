#!/bin/bash

check() {

    code=$?
    if [ $code -eq 0 ]; then 
        echo "OK" 
    else
        echo "Error" 
        exit 1
    fi 

}

# device
go test -v ./pkg/device/ -run Test2Ether
check

go test -v ./pkg/device/ -run TestEther
check

go test -v ./pkg/device/ -run TestNull
check

go test -v ./pkg/device/ -run TestLoopback
check

# ip
go test -v ./pkg/ip/ -run Test2
check 

go test -v ./pkg/ip/ -run TestIP
check

# other
go test -v ./pkg/ -run Test2
check

# test icmp,udp,tcp in manually for now 
# TODO: how to test icmp,udp,tcp automatically
