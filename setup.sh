#!/bin/bash

ip tuntap add mode tap user root name tap0
ip addr add 192.0.2.1/24 dev tap0
ip link set tap0 up

echo 1 > /proc/sys/net/ipv4/ip_forward
iptables -A FORWARD -o tap0 -j ACCEPT
iptables -A FORWARD -i tap0 -j ACCEPT
iptables -t nat -A POSTROUTING -s 192.0.2.0/24 -o eth0 -j MASQUERADE
