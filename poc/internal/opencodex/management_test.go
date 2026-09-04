package opencodex

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type managementCall struct {
	stdin string
	args  []string
}

type managementRunner struct {
	calls     []managementCall
	responses map[string]CommandResult
	errors    map[string]error
}

func (r *managementRunner) Run(_ context.Context, stdin string, args ...string) (CommandResult, error) {
	call := managementCall{stdin: stdin, args: append([]string(nil), args...)}
	r.calls = append(r.calls, call)
	key := managementCommandKey(args)
	if err := r.errors[key]; err != nil {
		return CommandResult{}, err
	}
	return r.responses[key], nil
}

type serviceManagementRunner struct {
	*managementRunner
	serviceCalls  []string
	serviceResult CommandResult
	serviceErr    error
}

func (r *serviceManagementRunner) RunServiceCommand(_ context.Context, action string) (CommandResult, error) {
	r.serviceCalls = append(r.serviceCalls, action)
	return r.serviceResult, r.serviceErr
}

func managementCommandKey(args []string) string {
	return strings.Join(args, "\x00")
}

func TestCombosParsesTargetsAndNormalizesNil(t *testing.T) {
	key := managementCommandKey([]string{"combo", "list", "--json"})
	runner := &managementRunner{responses: map[string]CommandResult{
		key: {Stdout: `{"combos":[{"id":"fallback","targets":[{"provider":"p1","model":"m1","weight":10}]}]}`},
	}}
	manager := NewManagerWithRunner("", runner, nil)
	combos, err := manager.Combos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Combo{{ID: "fallback", Targets: []ComboTarget{{Provider: "p1", Model: "m1"}}}}
	if !reflect.DeepEqual(combos, want) {
		t.Fatalf("Combos() = %#v, want %#v", combos, want)
	}

	runner.responses[key] = CommandResult{Stdout: `{"combos":null}`}
	combos, err = manager.Combos(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if combos == nil || len(combos) != 0 {
		t.Fatalf("Combos() = %#v, want non-nil empty slice", combos)
	}
}

func TestManagementCommandsUseExactPublicCLIArguments(t *testing.T) {
	cases := []struct {
		name string
		call func(*Manager) error
		args []string
	}{
		{name: "set default", call: func(m *Manager) error { return m.SetDefaultProvider(context.Background(), "p1") }, args: []string{"provider", "set-default", "p1", "--json"}},
		{name: "remove", call: func(m *Manager) error { return m.RemoveProvider(context.Background(), "p1") }, args: []string{"provider", "remove", "p1", "--json"}},
		{name: "start", call: func(m *Manager) error { return m.StartService(context.Background()) }, args: []string{"service", "start"}},
		{name: "stop", call: func(m *Manager) error { return m.StopService(context.Background()) }, args: []string{"service", "stop"}},
		{name: "port", call: func(m *Manager) error { return m.SetPort(context.Background(), 43123) }, args: []string{"config", "set", "port", "43123", "--json"}},
		{name: "update", call: func(m *Manager) error { return m.UpdateLatest(context.Background()) }, args: []string{"update", "--tag", "latest"}},
		{name: "sync", call: func(m *Manager) error { return m.SyncCatalogRestartCodex(context.Background()) }, args: []string{"sync", "--restart-codex"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			runner := &managementRunner{responses: map[string]CommandResult{managementCommandKey(test.args): {}}}
			if err := test.call(NewManagerWithRunner("", runner, nil)); err != nil {
				t.Fatal(err)
			}
			if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0].args, test.args) {
				t.Fatalf("calls = %#v, want args %#v", runner.calls, test.args)
			}
		})
	}
}

func TestRepairServiceUsesElevatedRunnerAndMapsUACFailure(t *testing.T) {
	runner := &serviceManagementRunner{
		managementRunner: &managementRunner{},
		serviceResult:    CommandResult{Stderr: "exit code 1223", Code: 1223},
	}
	err := NewManagerWithRunner("", runner, nil).RepairService(context.Background())
	if err == nil || !strings.Contains(err.Error(), "UAC") {
		t.Fatalf("RepairService() = %v, want UAC guidance", err)
	}
	if !reflect.DeepEqual(runner.serviceCalls, []string{"repair"}) {
		t.Fatalf("service calls = %v", runner.serviceCalls)
	}
	if len(runner.managementRunner.calls) != 0 {
		t.Fatalf("normal runner calls = %#v, want none", runner.managementRunner.calls)
	}
}

func TestRepairServiceUsesNormalRunnerWhenElevationIsUnavailable(t *testing.T) {
	args := []string{"service", "repair"}
	runner := &managementRunner{responses: map[string]CommandResult{managementCommandKey(args): {}}}
	if err := NewManagerWithRunner("", runner, nil).RepairService(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0].args, args) {
		t.Fatalf("calls = %#v, want %#v", runner.calls, args)
	}
}

func TestSetPortRejectsInvalidPortsWithoutRunningCLI(t *testing.T) {
	runner := &managementRunner{}
	manager := NewManagerWithRunner("", runner, nil)
	for _, port := range []int{0, -1, 65536} {
		if err := manager.SetPort(context.Background(), port); err == nil {
			t.Fatalf("SetPort(%d) error = nil", port)
		}
	}
	if len(runner.calls) != 0 {
		t.Fatalf("calls = %#v, want none", runner.calls)
	}
}

func TestManagementCommandsPropagateRunnerAndCommandErrors(t *testing.T) {
	runnerErr := errors.New("runner failed")
	runner := &managementRunner{errors: map[string]error{
		managementCommandKey([]string{"service", "start"}): runnerErr,
	}}
	if err := NewManagerWithRunner("", runner, nil).StartService(context.Background()); !errors.Is(err, runnerErr) {
		t.Fatalf("StartService() = %v, want %v", err, runnerErr)
	}

	args := []string{"provider", "remove", "p1", "--json"}
	runner = &managementRunner{responses: map[string]CommandResult{
		managementCommandKey(args): {Code: 9, Stderr: "remove failed"},
	}}
	err := NewManagerWithRunner("", runner, nil).RemoveProvider(context.Background(), "p1")
	if err == nil || !strings.Contains(err.Error(), "remove failed") {
		t.Fatalf("RemoveProvider() = %v, want command failure", err)
	}
}
