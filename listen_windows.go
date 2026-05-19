//go:build windows

package main

import "net"

func listenUDPReuse(addr string) (*net.UDPConn, error) {
	laddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp", laddr)
}
