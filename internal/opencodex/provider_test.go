package opencodex

import (
	"context"
	"testing"
)

type fakeRunner struct {
	calls []fakeCall
}

type fakeCall struct {
	stdin string
	args  []string
}

func (f *fakeRunner) Run(_ context.Context, stdin string, args ...string) (CommandResult, error) {
	f.calls = append(f.calls, fakeCall{stdin: stdin, args: append([]string(nil), args...)})
	if len(args) >= 3 && args[0] == "health" {
		return CommandResult{Stdout: `{"ok":true,"pid":12,"port":10123}`}, nil
	}
	if len(args) >= 3 && args[0] == "models" && args[1] == "list-custom" {
		return CommandResult{Stdout: `[]`}, nil
	}
	return CommandResult{}, nil
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

func TestNodeVersionAtLeast18(t *testing.T) {
	for value, want := range map[string]bool{"v18.0.0": true, "20.1.0": true, "v17.9.0": false, "unknown": false} {
		if got := nodeVersionAtLeast18(value); got != want {
			t.Fatalf("nodeVersionAtLeast18(%q) = %v, want %v", value, got, want)
		}
	}
}
