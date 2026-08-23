//go:build !windows

package store

import (
	"os/exec"
)

func setHideWindow(cmd *exec.Cmd) {
	// No-op on non-windows OS
}
