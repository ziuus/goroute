//go:build windows

package fast

import (
	"syscall"
)

func openRustLib(path string) (uintptr, error) {
	h, err := syscall.LoadLibrary(path)
	return uintptr(h), err
}
