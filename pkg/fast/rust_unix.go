//go:build !windows

package fast

import (
	"github.com/ebitengine/purego"
)

func openRustLib(path string) (uintptr, error) {
	return purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
}
