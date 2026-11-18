package execution

import (
	"os"
	"os/exec"

	"github.com/GTT-Community/gtt-cli/internal/ports"
)

// Start launches a process detached from this one and from the terminal (its own session, no stdin, stdout
// or stderr) and returns its pid without waiting. The child is released: it outlives the caller.
func (r Runner) Start(c ports.Command) (int, error) {
	cmd := exec.Command(c.Path, c.Args...)
	cmd.Dir = c.Dir
	if c.Env != nil {
		cmd.Env = append(os.Environ(), c.Env...)
	}
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	return pid, nil
}
