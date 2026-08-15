package azure

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

type fakeCLIRunner struct {
	installed bool
	account   string
	loginErr  error
	calls     [][]string
}

func (f *fakeCLIRunner) Run(_ context.Context, args ...string) (CLIResult, error) {
	f.calls = append(f.calls, append([]string(nil), args...))
	if !f.installed {
		return CLIResult{}, ErrAzureCLINotInstalled
	}
	if len(args) > 0 && args[0] == "version" {
		return CLIResult{Stdout: `{"azure-cli":"2.88.0"}`}, nil
	}
	if len(args) > 0 && args[0] == "login" {
		return CLIResult{}, f.loginErr
	}
	if len(args) >= 2 && args[0] == "account" && args[1] == "show" {
		if f.account == "" {
			return CLIResult{}, errors.New("not logged in")
		}
		return CLIResult{Stdout: f.account}, nil
	}
	return CLIResult{}, nil
}

func TestAuthStateReportsAzureCLIPrerequisites(t *testing.T) {
	for name, test := range map[string]struct {
		runner *fakeCLIRunner
		want   AuthState
	}{
		"missing": {
			runner: &fakeCLIRunner{},
			want:   AuthState{Message: "Azure CLI is not installed"},
		},
		"signed-out": {
			runner: &fakeCLIRunner{installed: true},
			want:   AuthState{CLIInstalled: true, CLIVersion: "2.88.0", Message: "Azure CLI sign-in is required"},
		},
		"signed-in": {
			runner: &fakeCLIRunner{installed: true, account: `{"tenantId":"tenant-1","user":{"name":"user@example.com"}}`},
			want:   AuthState{CLIInstalled: true, CLIVersion: "2.88.0", SignedIn: true, Username: "user@example.com", TenantID: "tenant-1", Message: "Azure CLI is signed in"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			client := NewSDKClientWithRunner(test.runner)
			if got := client.AuthState(context.Background()); got != test.want {
				t.Fatalf("AuthState() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestAuthenticateRunsAzureCLILoginAndActivatesSDKClient(t *testing.T) {
	runner := &fakeCLIRunner{installed: true, account: `{"tenantId":"tenant-1","user":{"name":"user@example.com"}}`}
	client := NewSDKClientWithRunner(runner)

	state, err := client.Authenticate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !state.SignedIn || client.ready() != nil {
		t.Fatalf("Azure CLI account was not activated: %#v", state)
	}
	wantLogin := []string{"login", "--allow-no-subscriptions", "--output", "none", "--only-show-errors"}
	found := false
	for _, call := range runner.calls {
		if slices.Equal(call, wantLogin) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("az login call not found: %v", runner.calls)
	}
}

func TestAuthenticateDoesNotExposeAzureCLIErrorOutput(t *testing.T) {
	runner := &fakeCLIRunner{installed: true, loginErr: errors.New("access-token-value")}
	client := NewSDKClientWithRunner(runner)

	_, err := client.Authenticate(context.Background())
	if err == nil || strings.Contains(err.Error(), "access-token-value") {
		t.Fatalf("Azure CLI error leaked command details: %v", err)
	}
}

func TestLiveAzureCLIState(t *testing.T) {
	if os.Getenv("FOUNDRYCODEX_LIVE_AZURE_CLI") != "1" {
		t.Skip("set FOUNDRYCODEX_LIVE_AZURE_CLI=1 to inspect the local Azure CLI")
	}
	state := NewSDKClient().AuthState(context.Background())
	if !state.CLIInstalled || state.CLIVersion == "" {
		t.Fatal("Azure CLI version was not detected")
	}
	t.Logf("Azure CLI %s detected; signedIn=%v", state.CLIVersion, state.SignedIn)
}
