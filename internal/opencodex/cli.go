package opencodex

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type LocalRunner struct {
	dataDir string
}

func NewLocalRunner(dataDir string) *LocalRunner {
	return &LocalRunner{dataDir: dataDir}
}

func (r *LocalRunner) Run(ctx context.Context, stdin string, args ...string) (CommandResult, error) {
	path, err := r.commandPath()
	if err != nil {
		return CommandResult{}, err
	}
	command := exec.CommandContext(ctx, path, args...)
	if stdin != "" {
		command.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	result := CommandResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err == nil {
		return result, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.Code = exitError.ExitCode()
	}
	return result, redactError(err, stdin, result.Stderr)
}

func (r *LocalRunner) commandPath() (string, error) {
	if path, err := exec.LookPath("ocx"); err == nil {
		return path, nil
	}
	managedCandidates := []string{
		filepath.Join(r.dataDir, "opencodex", "node_modules", ".bin", "ocx.cmd"),
		filepath.Join(r.dataDir, "opencodex", "node_modules", ".bin", "ocx"),
	}
	for _, candidate := range managedCandidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("ocx was not found on PATH or in the managed opencodex directory")
}

func (r *LocalRunner) NodeCommand(ctx context.Context, name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s was not found on PATH", name)
	}
	result := exec.CommandContext(ctx, path, "--version")
	output, err := result.Output()
	if err != nil {
		return "", fmt.Errorf("%s version check failed", name)
	}
	return strings.TrimSpace(string(output)), nil
}

func redactError(err error, secret, stderr string) error {
	message := err.Error()
	if stderr != "" {
		message += ": " + stderr
	}
	message = redactSecret(message, secret)
	return errors.New(message)
}

func redactSecret(message, secret string) string {
	if secret == "" {
		return message
	}
	message = strings.ReplaceAll(message, secret, "[REDACTED]")
	if trimmed := strings.TrimSpace(secret); trimmed != "" {
		message = strings.ReplaceAll(message, trimmed, "[REDACTED]")
	}
	return message
}

func platformCommandName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".cmd"
	}
	return name
}
