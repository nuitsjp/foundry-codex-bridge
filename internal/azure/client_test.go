package azure

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices/v4"
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

func TestCredentialOptionsForScopeUseEitherTenantOrSubscription(t *testing.T) {
	tests := []struct {
		name           string
		tenantID       string
		subscriptionID string
		wantTenant     string
		wantSub        string
	}{
		{name: "tenant scope", tenantID: " tenant-1 ", wantTenant: "tenant-1"},
		{name: "subscription scope", tenantID: "tenant-1", subscriptionID: " subscription-1 ", wantSub: "subscription-1"},
		{name: "default scope"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := credentialOptionsForScope(test.tenantID, test.subscriptionID)
			if options.TenantID != test.wantTenant || options.Subscription != test.wantSub {
				t.Fatalf("credential options = tenant %q, subscription %q; want tenant %q, subscription %q", options.TenantID, options.Subscription, test.wantTenant, test.wantSub)
			}
			if options.TenantID != "" && options.Subscription != "" {
				t.Fatal("Azure CLI credential must not receive tenant and subscription together")
			}
		})
	}
}

func TestModelResourceGroupsFilterAndDeduplicateAccounts(t *testing.T) {
	accounts := []*armcognitiveservices.Account{
		accountWithResourceID("AIServices", "/subscriptions/sub-1/resourceGroups/rg-zeta/providers/Microsoft.CognitiveServices/accounts/foundry-1"),
		accountWithResourceID("OpenAI", "/subscriptions/sub-1/resourceGroups/rg-alpha/providers/Microsoft.CognitiveServices/accounts/openai-1"),
		accountWithResourceID("OpenAI", "/subscriptions/sub-1/resourceGroups/RG-ZETA/providers/Microsoft.CognitiveServices/accounts/openai-2"),
		accountWithResourceID("TextAnalytics", "/subscriptions/sub-1/resourceGroups/rg-unsupported/providers/Microsoft.CognitiveServices/accounts/language-1"),
		accountWithResourceID("AIServices", "not-an-azure-resource-id"),
		nil,
	}

	got := modelResourceGroups(accounts)
	want := []ResourceGroup{{Name: "rg-alpha"}, {Name: "rg-zeta"}}
	if !slices.Equal(got, want) {
		t.Fatalf("modelResourceGroups() = %#v, want %#v", got, want)
	}
}

func accountWithResourceID(kind, id string) *armcognitiveservices.Account {
	return &armcognitiveservices.Account{Kind: &kind, ID: &id}
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
