package opencodex

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls      []fakeCall
	statusJSON string
}

type fakeCall struct {
	stdin string
	args  []string
}

func (f *fakeRunner) Run(_ context.Context, stdin string, args ...string) (CommandResult, error) {
	f.calls = append(f.calls, fakeCall{stdin: stdin, args: append([]string(nil), args...)})
	if len(args) >= 1 && args[0] == "status" {
		return CommandResult{Stdout: f.statusJSON}, nil
	}
	if len(args) >= 1 && args[0] == "health" {
		return CommandResult{Stdout: `{"ok":true,"pid":12,"port":10123}`}, nil
	}
	if len(args) >= 2 && args[0] == "models" && args[1] == "list-custom" {
		return CommandResult{Stdout: `[]`}, nil
	}
	return CommandResult{}, nil
}

type leakingRunner struct{ secret string }

func (r leakingRunner) Run(context.Context, string, ...string) (CommandResult, error) {
	return CommandResult{}, errors.New("command failed for " + r.secret)
}

func TestSyncCommandsDoNotPutPrimaryKeyInArguments(t *testing.T) {
	runner := &fakeRunner{}
	manager := NewManagerWithRunner("C:\\Local", runner, nil)
	secret := "primary-secret-value"
	if err := manager.AddPrimaryKey(context.Background(), "az-account", secret); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(runner.calls))
	}
	call := runner.calls[0]
	for _, arg := range call.args {
		if arg == secret {
			t.Fatalf("secret was passed as an argument")
		}
	}
	if call.stdin != secret+"\n" {
		t.Fatalf("secret was not passed through stdin")
	}
}

func TestAddPrimaryKeyRedactsRunnerErrors(t *testing.T) {
	secret := "primary-secret-value"
	manager := NewManagerWithRunner("C:\\Local", leakingRunner{secret: secret}, nil)
	err := manager.AddPrimaryKey(context.Background(), "az-account", secret)
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("secret leaked in error: %v", err)
	}
}

func TestRedactErrorRemovesSecretWithoutTrailingNewline(t *testing.T) {
	err := redactError(errors.New("exit status 1"), "primary-secret\n", "failed for primary-secret")
	if strings.Contains(err.Error(), "primary-secret") {
		t.Fatalf("secret leaked in error: %v", err)
	}
}

func TestEnsureProviderDisablesLiveModels(t *testing.T) {
	runner := &fakeRunner{}
	manager := NewManagerWithRunner("C:\\Local", runner, nil)
	if err := manager.EnsureProvider(context.Background(), "az-account", "https://account.openai.azure.com/openai", "gpt-4o"); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("got %d calls, want provider add and edit", len(runner.calls))
	}
	want := []string{"provider", "edit", "az-account", "--live-models", "off", "--json"}
	if !slices.Equal(runner.calls[1].args, want) {
		t.Fatalf("provider edit args = %v, want %v", runner.calls[1].args, want)
	}
}

func TestEnsureServiceUsesStatusToChooseAction(t *testing.T) {
	for name, test := range map[string]struct {
		status string
		want   []string
	}{
		"healthy":  {status: `{"startup":{"serviceInstalled":true,"serviceViable":true,"serviceConflict":false}}`},
		"absent":   {status: `{"startup":{"serviceInstalled":false,"serviceViable":false,"serviceConflict":false}}`, want: []string{"service", "install"}},
		"stopped":  {status: `{"startup":{"serviceInstalled":true,"serviceViable":false,"serviceConflict":false}}`, want: []string{"service", "repair"}},
		"conflict": {status: `{"startup":{"serviceInstalled":true,"serviceViable":false,"serviceConflict":true}}`, want: []string{"service", "install"}},
	} {
		t.Run(name, func(t *testing.T) {
			runner := &fakeRunner{statusJSON: test.status}
			manager := NewManagerWithRunner("C:\\Local", runner, nil)
			if err := manager.EnsureService(context.Background()); err != nil {
				t.Fatal(err)
			}
			if len(runner.calls) != 1+boolInt(len(test.want) > 0) {
				t.Fatalf("calls = %v", runner.calls)
			}
			if len(test.want) > 0 && !slices.Equal(runner.calls[1].args, test.want) {
				t.Fatalf("service args = %v, want %v", runner.calls[1].args, test.want)
			}
		})
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestNodeVersionAtLeast18(t *testing.T) {
	for value, want := range map[string]bool{"v18.0.0": true, "20.1.0": true, "v17.9.0": false, "unknown": false} {
		if got := nodeVersionAtLeast18(value); got != want {
			t.Fatalf("nodeVersionAtLeast18(%q) = %v, want %v", value, got, want)
		}
	}
}
