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

	"github.com/nuitsjp/foundry-codex-bridge/internal/platform"
)

type LocalRunner struct {
	dataDir string
}

type commandSpec struct {
	path   string
	prefix []string
}

func NewLocalRunner(dataDir string) *LocalRunner {
	return &LocalRunner{dataDir: dataDir}
}

func (r *LocalRunner) Run(ctx context.Context, stdin string, args ...string) (CommandResult, error) {
	spec, err := r.command()
	if err != nil {
		return CommandResult{}, err
	}
	commandArgs := append(append([]string(nil), spec.prefix...), args...)
	command := exec.CommandContext(ctx, spec.path, commandArgs...)
	platform.HideCommandWindow(command)
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

func (r *LocalRunner) RunServiceCommand(ctx context.Context, action string) (CommandResult, error) {
	spec, err := r.command()
	if err != nil {
		return CommandResult{}, err
	}
	workingDir := ""
	if len(spec.prefix) > 0 {
		workingDir = filepath.Dir(filepath.Dir(spec.prefix[0]))
	}
	code, err := platform.RunElevatedCommand(ctx, spec.path, workingDir, append(spec.prefix, "service", action)...)
	return CommandResult{Code: code}, err
}

func (r *LocalRunner) command() (commandSpec, error) {
	nodePath, nodeErr := exec.LookPath("node")
	if path, err := exec.LookPath("ocx"); err == nil {
		if !isCommandShim(path) {
			return commandSpec{path: path}, nil
		}
		if nodeErr == nil {
			entry := commandShimEntry(path)
			if _, err := os.Stat(entry); err == nil {
				return commandSpec{path: nodePath, prefix: []string{entry}}, nil
			}
		}
	}

	managedEntry := filepath.Join(r.dataDir, "opencodex", "node_modules", "@bitkyc08", "opencodex", "bin", "ocx.mjs")
	if nodeErr == nil {
		if _, err := os.Stat(managedEntry); err == nil {
			return commandSpec{path: nodePath, prefix: []string{managedEntry}}, nil
		}
	}
	return commandSpec{}, errors.New("ocx was not found on PATH or in the managed opencodex directory")
}

func isCommandShim(path string) bool {
	return runtime.GOOS == "windows" && strings.EqualFold(filepath.Ext(path), ".cmd")
}

func commandShimEntry(path string) string {
	directory := filepath.Dir(path)
	if strings.EqualFold(filepath.Base(directory), ".bin") {
		directory = filepath.Dir(directory)
	} else {
		directory = filepath.Join(directory, "node_modules")
	}
	return filepath.Join(directory, "@bitkyc08", "opencodex", "bin", "ocx.mjs")
}

func (r *LocalRunner) NodeCommand(ctx context.Context, name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s was not found on PATH", name)
	}
	result := exec.CommandContext(ctx, path, "--version")
	platform.HideCommandWindow(result)
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
