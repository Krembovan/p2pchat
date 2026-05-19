//go:build linux

package main

import "syscall"

func setReusePort(fd uintptr) {
	syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, 0x0F, 1)
}
