# Go-network

Go-network is TCP/IP protocol stack written in Go.

## Feature

- device
    - loopback
    - null
    - ethernet
        - tap device

- IP(v4)
- ARP
- ICMP
- UDP
- TCP

## Setup

You can set up this protocol stack like this. Some examples are in `go-network/example`
directory.

```
git clone https://github.com/hedwig100/go-network && cd go-network
docker build -t go-network .
docker run -it --name go-network --privileged go-network /bin/bash # privileged access is necessary for tap device
./setup.sh
```

Examples in `go-network/example` work like this.

-  ICMP
```
go run example/icmp/main.go&
ping 192.0.2.2
```
- UDP
```
go run example/udp/main.go&
nc -u 192.0.2.2 8080
```
- TCP
```
go run example/tcp/main.go&
nc -nv 192.0.2.2 8080
```

## Log Example
- ARP resolve
```
2022/02/19 08:13:13 [D] Ether rxHandler: dev=tap0,protocolType=ARP,len=42,header=
		Src: ff:ff:ff:ff:ff:ff,
		Dst: 5a:b0:52:6d:15:84,
		Type: ARP
	
2022/02/19 08:13:13 [I] input data dev=5a:b0:52:6d:15:84,typ=ARP,data:[0 1 8 0 6 4 0 1 90 176 82 109 21 132 192 0 2 1 0 0 0 0 0 0 192 0 2 2]
2022/02/19 08:13:13 [D] ARP cache insert pa=192.0.2.1,ha=5a:b0:52:6d:15:84,timeval=2022-02-19 08:13:13.3629771 +0000 UTC m=+6.964047501
2022/02/19 08:13:13 [D] ARP rxHandler: dev=tap0,arp header=
		Hrd: Ethernet(1),
		Pro: IPv4(2048),
		Hln: 6,
		Pln: 4,
		Op: Request(1),
		Sha: 5a:b0:52:6d:15:84,
		Spa: 192.0.2.1,
		Tha: 0:0:0:0:0:0,
		Tpa: 192.0.2.2,
	
2022/02/19 08:13:13 [D] ARP TxHandler(reply): dev=tap0,arp header=
		Hrd: Ethernet(1),
		Pro: IPv4(2048),
		Hln: 6,
		Pln: 4,
		Op: Reply(2),
		Sha: 5a:b0:52:6d:15:84,
		Spa: 192.0.2.2,
		Tha: 5a:b0:52:6d:15:84,
		Tpa: 192.0.2.1,
	
2022/02/19 08:13:13 ARP 5a:b0:52:6d:15:84
2022/02/19 08:13:13 [D] Ether TxHandler: data is trasmitted by ethernet-device(name=tap0),header=
		Src: 5a:b0:52:6d:15:84,
		Dst: 5a:b0:52:6d:15:84,
		Type: ARP
	
2022/02/19 08:13:13 [I] device output, dev=tap0,typ=ARP
```
<br>

- ICMP echo-reply

```
2022/02/19 08:14:21 [D] Ether rxHandler: dev=tap0,protocolType=IPv4,len=98,header=
		Src: 5a:b0:52:6d:15:84,
		Dst: 5a:b0:52:6d:15:84,
		Type: IPv4
	
2022/02/19 08:14:23 [I] input data dev=5a:b0:52:6d:15:84,typ=IPv4,data:[69 0 0 84 222 161 64 0 64 1 216 3 192 0 2 1 192 0 2 2 8 0 166 22 100 210 0 3 223 166 16 98 0 0 0 0 52 56 10 0 0 0 0 0 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31 32 33 34 35 36 37 38 39 40 41 42 43 44 45 46 47 48 49 50 51 52 53 54 55]
2022/02/19 08:14:23 [D] IP rxHandler: iface=192.0.2.2,protocol=ICMP,header=
		Version: 4,
		Header Length: 20,
		Total Length: 84,
		Tos: 0,
		Id: 56993,
		Flags: 2,
		Fragment Offset: 0,
		TTL: 64,
		ProtocolType: ICMP,
		Checksum: d803,
		Src: 192.0.2.1,
		Dst: 192.0.2.2,
	
2022/02/19 08:14:23 [D] ICMP rxHandler: iface=1,header=
		typ: ICMPTypeEcho, 
		code: UNKNOWN(0),
		checksum: a616,
		id: 25810,
		seq: 3,
	
2022/02/19 08:14:23 [D] ICMP TxHanlder: 192.0.2.2 => 192.0.2.1,header=
		typ: ICMPTypeEchoReply, 
		code: UNKNOWN(0),
		checksum: 9b2a,
		id: 25810,
		seq: 3,
	
2022/02/19 08:14:23 [D] IP TxHandler: iface=1,dev=tap0,header=
		Version: 4,
		Header Length: 20,
		Total Length: 84,
		Tos: 0,
		Id: 3,
		Flags: 0,
		Fragment Offset: 0,
		TTL: 255,
		ProtocolType: ICMP,
		Checksum: 37a2,
		Src: 192.0.2.2,
		Dst: 192.0.2.1,
	
2022/02/19 08:14:23 IPv4 5a:b0:52:6d:15:84
2022/02/19 08:14:23 [D] Ether TxHandler: data is trasmitted by ethernet-device(name=tap0),header=
		Src: 5a:b0:52:6d:15:84,
		Dst: 5a:b0:52:6d:15:84,
		Type: IPv4
	

```
<br>

- Threeway hand shake

```
2022/02/19 08:13:13 [D] Ether rxHandler: dev=tap0,protocolType=IPv4,len=74,header=
		Src: 5a:b0:52:6d:15:84,
		Dst: 5a:b0:52:6d:15:84,
		Type: IPv4
	
2022/02/19 08:13:13 [I] input data dev=5a:b0:52:6d:15:84,typ=IPv4,data:[69 0 0 60 75 135 64 0 64 6 107 49 192 0 2 1 192 0 2 2 164 250 31 144 164 220 198 36 0 0 0 0 160 2 250 240 113 3 0 0 2 4 5 180 4 2 8 10 245 31 51 92 0 0 0 0 1 3 3 7]
2022/02/19 08:13:13 [D] IP rxHandler: iface=192.0.2.2,protocol=TCP,header=
		Version: 4,
		Header Length: 20,
		Total Length: 60,
		Tos: 0,
		Id: 19335,
		Flags: 2,
		Fragment Offset: 0,
		TTL: 64,
		ProtocolType: TCP,
		Checksum: 6b31,
		Src: 192.0.2.1,
		Dst: 192.0.2.2,
	
2022/02/19 08:13:13 [D] TCP rxHandler: src=192.0.2.1:42234,dst=192.0.2.2:8080,iface=IPv4,len=0,tcp header=
		Dst: 8080, 
		Src: 42234,
		Seq: 2765932068, 
		Ack: 0,
		Offset: 10,
		Control Flag: SYN ,
		Window Size: 64240,
		Checksum: 7103,
		Urgent Pointer: 0,
	,payload=[2 4 5 180 4 2 8 10 245 31 51 92 0 0 0 0 1 3 3 7]
2022/02/19 08:13:13 [I] local=192.0.2.2:8080, LISTEN => SYN-RECEIVED
2022/02/19 08:13:13 [D] TCP TxHandler: src=192.0.2.2:8080,dst=192.0.2.1:42234,len=0,tcp header=
		Dst: 42234, 
		Src: 8080,
		Seq: 1976163873, 
		Ack: 2765932069,
		Offset: 5,
		Control Flag: ACK SYN ,
		Window Size: 65535,
		Checksum: a856,
		Urgent Pointer: 0,
	
2022/02/19 08:13:13 [D] IP TxHandler: iface=1,dev=tap0,header=
		Version: 4,
		Header Length: 20,
		Total Length: 40,
		Tos: 0,
		Id: 1,
		Flags: 0,
		Fragment Offset: 0,
		TTL: 255,
		ProtocolType: TCP,
		Checksum: 37cb,
		Src: 192.0.2.2,
		Dst: 192.0.2.1,
	
2022/02/19 08:13:13 IPv4 5a:b0:52:6d:15:84
2022/02/19 08:13:13 [D] Ether TxHandler: data is trasmitted by ethernet-device(name=tap0),header=
		Src: 5a:b0:52:6d:15:84,
		Dst: 5a:b0:52:6d:15:84,
		Type: IPv4
	
2022/02/19 08:13:13 [D] Ether rxHandler: dev=tap0,protocolType=IPv4,len=54,header=
		Src: 5a:b0:52:6d:15:84,
		Dst: 5a:b0:52:6d:15:84,
		Type: IPv4
	
2022/02/19 08:13:13 [I] input data dev=5a:b0:52:6d:15:84,typ=IPv4,data:[69 0 0 40 75 136 64 0 64 6 107 68 192 0 2 1 192 0 2 2 164 250 31 144 164 220 198 37 117 201 222 34 80 16 250 240 173 102 0 0]
2022/02/19 08:13:13 [D] IP rxHandler: iface=192.0.2.2,protocol=TCP,header=
		Version: 4,
		Header Length: 20,
		Total Length: 40,
		Tos: 0,
		Id: 19336,
		Flags: 2,
		Fragment Offset: 0,
		TTL: 64,
		ProtocolType: TCP,
		Checksum: 6b44,
		Src: 192.0.2.1,
		Dst: 192.0.2.2,
	
2022/02/19 08:13:13 [D] TCP rxHandler: src=192.0.2.1:42234,dst=192.0.2.2:8080,iface=IPv4,len=0,tcp header=
		Dst: 8080, 
		Src: 42234,
		Seq: 2765932069, 
		Ack: 1976163874,
		Offset: 5,
		Control Flag: ACK ,
		Window Size: 64240,
		Checksum: ad66,
		Urgent Pointer: 0,
	,payload=[]
2022/02/19 08:13:13 [I] local=192.0.2.2:8080, SYN-RECEIVED => ESTABLISHED
2022/02/19 08:13:13 [I] RTT=2.1525ms,RTO=11.901097775s
```

## License
This software is released under the MIT License, see LICENSE.