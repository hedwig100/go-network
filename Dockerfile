FROM golang:1.16.13-bullseye
RUN apt-get update && apt install -y \
    iproute2 \
    iputils-ping \
    netcat-openbsd \
    iptables

COPY . /go/src/go-network
WORKDIR /go/src/go-network