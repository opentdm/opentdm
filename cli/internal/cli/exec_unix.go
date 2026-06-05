//go:build !windows

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// runProcess replaces the current process image with the target command
// (syscall.Exec), inheriting the controlling TTY for correct signal handling
// and exit-code propagation. It only returns on failure to start.
func runProcess(argv, env []string) int {
	path, err := exec.LookPath(argv[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "opentdm: %v\n", err)
		return 127
	}
	if err := syscall.Exec(path, argv, env); err != nil {
		fmt.Fprintf(os.Stderr, "opentdm: exec %s: %v\n", argv[0], err)
		return 126
	}
	return 0 // unreachable on success
}
