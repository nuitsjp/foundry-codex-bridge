package opencodex

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls         []fakeCall
	statusJSON    string
	accountErr    error
	providersJSON string
	providerJSON  string
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
	if len(args) >= 3 && args[0] == "provider" && args[1] == "list" {
		payload := f.providersJSON
		if payload == "" {
			payload = `{"configured":[]}`
		}
		return CommandResult{Stdout: payload}, nil
	}
	if len(args) >= 3 && args[0] == "provider" && args[1] == "show" {
		return CommandResult{Stdout: f.providerJSON}, nil
	}
	if len(args) >= 2 && args[0] == "account" && args[1] == "list" && f.accountErr != nil {
		err := f.accountErr
		f.accountErr = nil
		return CommandResult{Stderr: err.Error(), Code: 1}, err
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
	if len(runner.calls) != 4 {
		t.Fatalf("got %d calls, want provider list, add, edit, and account readiness", len(runner.calls))
	}
	want := []string{"provider", "edit", "az-account", "--live-models", "off", "--json"}
	if !slices.Equal(runner.calls[2].args, want) {
		t.Fatalf("provider edit args = %v, want %v", runner.calls[2].args, want)
	}
	want = []string{"account", "list", "az-account", "--json"}
	if !slices.Equal(runner.calls[3].args, want) {
		t.Fatalf("account readiness args = %v, want %v", runner.calls[3].args, want)
	}
}

func TestEnsureProviderSkipsMutationWhenConfigurationMatches(t *testing.T) {
	runner := &fakeRunner{
		providersJSON: `{"configured":[{"name":"az-account"}]}`,
		providerJSON:  `{"name":"az-account","adapter":"azure-openai","baseUrl":"https://account.openai.azure.com/openai","defaultModel":"gpt-4o","liveModels":false}`,
	}
	manager := NewManagerWithRunner("C:\\Local", runner, nil)
	if err := manager.EnsureProvider(context.Background(), "az-account", "https://account.openai.azure.com/openai", "gpt-4o"); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 2 {
		t.Fatalf("calls = %v, want provider list and show only", runner.calls)
	}
}

func TestEnsureProviderWaitsUntilLiveProxyRecognizesProvider(t *testing.T) {
	runner := &fakeRunner{accountErr: errors.New("Error: unknown provider")}
	manager := NewManagerWithRunner("C:\\Local", runner, nil)
	if err := manager.EnsureProvider(context.Background(), "az-account", "https://account.openai.azure.com/openai", "gpt-4o"); err != nil {
		t.Fatal(err)
	}
	accountCalls := 0
	for _, call := range runner.calls {
		if len(call.args) >= 2 && call.args[0] == "account" && call.args[1] == "list" {
			accountCalls++
		}
	}
	if accountCalls != 2 {
		t.Fatalf("account readiness calls = %d, want 2", accountCalls)
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

func TestServiceCommandErrorExplainsWindowsApprovalFailure(t *testing.T) {
	tests := []struct {
		name   string
		result CommandResult
		err    error
	}{
		{name: "truncated UAC exit code", err: errors.New("exit status 1: Background service install failed with exit code 199")},
		{name: "Windows cancellation code", result: CommandResult{Stderr: "Background service install failed with exit code 1223"}},
		{name: "direct Windows cancellation code", err: errors.New("Windows elevation failed (code 1223): The operation was canceled by the user")},
		{name: "elevated process cancellation code", result: CommandResult{Code: 1223}},
		{name: "Task Scheduler marker", result: CommandResult{Stderr: "OCX_ERROR_CODE=WINDOWS_SCHTASKS_CREATE_ACCESS_DENIED"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := serviceCommandError("ocx service install", test.result, test.err)
			if !strings.Contains(err.Error(), "ユーザー アカウント制御（UAC）を承認") {
				t.Fatalf("serviceCommandError() = %q", err)
			}
		})
	}
}

func TestServiceCommandErrorPreservesUnclassifiedFailure(t *testing.T) {
	want := errors.New("unclassified failure")
	if got := serviceCommandError("ocx service install", CommandResult{}, want); !errors.Is(got, want) {
		t.Fatalf("serviceCommandError() = %v, want %v", got, want)
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
