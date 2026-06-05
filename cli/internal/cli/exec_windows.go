//go:build windows

package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// runProcess runs the target command as a child, forwarding stdio and
// propagating the child's exit code (Windows has no execve).
func runProcess(argv, env []string) int {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return ee.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "opentdm: %v\n", err)
		return 1
	}
	return 0
}
