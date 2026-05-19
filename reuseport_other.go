//go:build !linux && !darwin && !freebsd

package main

func setReusePort(fd uintptr) {
}
