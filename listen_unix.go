//go:build linux || darwin || freebsd

package main

import (
	"net"
	"syscall"
)

func listenUDPReuse(addr string) (*net.UDPConn, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				setReusePort(fd)
			})
		},
	}
	pc, err := lc.ListenPacket(nil, "udp", addr)
	if err != nil {
		return nil, err
	}
	return pc.(*net.UDPConn), nil
}
