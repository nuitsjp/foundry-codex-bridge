//go:build !windows

package platform

import (
	"context"
	"os/exec"
)

func HideCommandWindow(_ *exec.Cmd) {}

func RunElevatedCommand(ctx context.Context, path, workingDir string, args ...string) (int, error) {
	command := exec.CommandContext(ctx, path, args...)
	command.Dir = workingDir
	err := command.Run()
	if exitError, ok := err.(*exec.ExitError); ok {
		return exitError.ExitCode(), nil
	}
	return 0, err
}
