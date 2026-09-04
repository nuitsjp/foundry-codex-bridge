package opencodex

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
)

func (m *Manager) Combos(ctx context.Context) ([]Combo, error) {
	result, err := m.runner.Run(ctx, "", "combo", "list", "--json")
	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, commandFailure("ocx combo list", result)
	}
	var payload struct {
		Combos []Combo `json:"combos"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		return nil, errors.New("ocx combo list returned invalid JSON")
	}
	if payload.Combos == nil {
		payload.Combos = []Combo{}
	}
	return payload.Combos, nil
}

func (m *Manager) SetDefaultProvider(ctx context.Context, providerID string) error {
	return m.runManagementCommand(ctx, "ocx provider set-default", "provider", "set-default", providerID, "--json")
}

func (m *Manager) RemoveProvider(ctx context.Context, providerID string) error {
	return m.runManagementCommand(ctx, "ocx provider remove", "provider", "remove", providerID, "--json")
}

func (m *Manager) StartService(ctx context.Context) error {
	return m.runManagementCommand(ctx, "ocx service start", "service", "start")
}

func (m *Manager) StopService(ctx context.Context) error {
	return m.runManagementCommand(ctx, "ocx service stop", "service", "stop")
}

func (m *Manager) RepairService(ctx context.Context) error {
	if serviceRunner, ok := m.runner.(interface {
		RunServiceCommand(context.Context, string) (CommandResult, error)
	}); ok {
		result, err := serviceRunner.RunServiceCommand(ctx, "repair")
		if err != nil || result.Code != 0 {
			return serviceCommandError("ocx service repair", result, err)
		}
		return nil
	}
	return m.runManagementCommand(ctx, "ocx service repair", "service", "repair")
}

func (m *Manager) SetPort(ctx context.Context, port int) error {
	if port < 1 || port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	return m.runManagementCommand(ctx, "ocx config set port", "config", "set", "port", strconv.Itoa(port), "--json")
}

func (m *Manager) UpdateLatest(ctx context.Context) error {
	return m.runManagementCommand(ctx, "ocx update", "update", "--tag", "latest")
}

func (m *Manager) SyncCatalogRestartCodex(ctx context.Context) error {
	return m.runManagementCommand(ctx, "ocx sync", "sync", "--restart-codex")
}

func (m *Manager) runManagementCommand(ctx context.Context, command string, args ...string) error {
	result, err := m.runner.Run(ctx, "", args...)
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return commandFailure(command, result)
	}
	return nil
}

var _ ManagerAPI = (*Manager)(nil)
