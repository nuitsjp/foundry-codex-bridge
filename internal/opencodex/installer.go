package opencodex

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNodePrerequisite = errors.New("Node.js 18 or newer and npm are required")

func (m *Manager) nodeInfo(ctx context.Context) NodeInfo {
	result := NodeInfo{RequiredNode: ">=18"}
	if version, err := commandVersion(ctx, "node"); err == nil {
		result.NodeInstalled = true
		result.NodeVersion = version
	}
	if version, err := commandVersion(ctx, "npm"); err == nil {
		result.NpmInstalled = true
		result.NpmVersion = version
	}
	result.Compatible = result.NodeInstalled && result.NpmInstalled && nodeVersionAtLeast18(result.NodeVersion)
	return result
}

func (m *Manager) prepare(ctx context.Context) (State, error) {
	info := m.nodeInfo(ctx)
	if !info.NodeInstalled || !info.NpmInstalled || !nodeVersionAtLeast18(info.NodeVersion) {
		return State{NodeInfo: info, Message: ErrNodePrerequisite.Error()}, ErrNodePrerequisite
	}
	installDir := filepath.Join(m.dataDir, "opencodex")
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		return State{NodeInfo: info, Message: "opencodex directory could not be created"}, err
	}
	result, err := m.runNPM(ctx, "install", "--prefix", installDir, "--no-save", "@bitkyc08/opencodex@latest")
	if err != nil {
		return State{NodeInfo: info, Message: "opencodex installation failed"}, err
	}
	if result.Code != 0 {
		return State{NodeInfo: info, Message: "opencodex installation failed"}, fmt.Errorf("npm install failed: %s", strings.TrimSpace(result.Stderr))
	}
	return m.state(ctx)
}

func (m *Manager) runNPM(ctx context.Context, args ...string) (CommandResult, error) {
	path, err := m.findCommand("npm")
	if err != nil {
		return CommandResult{}, err
	}
	return runCommand(ctx, path, "", args...)
}

func (m *Manager) findCommand(name string) (string, error) {
	return findCommandOnPath(name)
}

func findCommandOnPath(name string) (string, error) {
	for _, candidate := range []string{name, platformCommandName(name)} {
		if path, err := execLookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s was not found on PATH", name)
}

func nodeVersionAtLeast18(value string) bool {
	value = strings.TrimSpace(strings.TrimPrefix(value, "v"))
	var major int
	if _, err := fmt.Sscanf(value, "%d", &major); err != nil {
		return false
	}
	return major >= 18
}

func commandVersion(ctx context.Context, name string) (string, error) {
	path, err := findCommandOnPath(name)
	if err != nil {
		return "", err
	}
	output, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
