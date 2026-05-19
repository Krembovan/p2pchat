//go:build darwin || freebsd

package main

import "syscall"

func setReusePort(fd uintptr) {
	syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
}
