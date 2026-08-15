package azure

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
)

var ErrAzureCLINotInstalled = errors.New("Azure CLI is not installed")

type CLIResult struct {
	Stdout string
	Stderr string
}

type CLIRunner interface {
	Run(context.Context, ...string) (CLIResult, error)
}

type localCLIRunner struct{}

func (localCLIRunner) Run(ctx context.Context, args ...string) (CLIResult, error) {
	path, err := exec.LookPath("az")
	if err != nil {
		return CLIResult{}, ErrAzureCLINotInstalled
	}
	command := exec.CommandContext(ctx, path, args...)
	command.Env = append(os.Environ(), "AZURE_CORE_LOGIN_EXPERIENCE_V2=off")
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	return CLIResult{Stdout: stdout.String(), Stderr: stderr.String()}, err
}
