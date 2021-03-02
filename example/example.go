package main

import (
	"net"
)

func main() {
	remote, err := net.ResolveUDPAddr("udp4", "10.28.0.73:61506")
	if err != nil {
		panic(err)
	}
	conn, err := net.DialUDP("udp4", nil, remote)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	_, err = conn.Write([]byte("Привет мир!"))
	if err != nil {
		panic(err)
	}
}
