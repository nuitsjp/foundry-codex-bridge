package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/nuitsjp/foundry-codex-bridge/internal/azure"
	"github.com/nuitsjp/foundry-codex-bridge/internal/opencodex"
)

type fakeAzure struct {
	resource azure.ModelResource
	keys     azure.KeyBundle
}

func (f *fakeAzure) Authenticate(context.Context) (azure.AuthState, error) {
	return azure.AuthState{SignedIn: true}, nil
}
func (f *fakeAzure) SignOut(context.Context) error                   { return nil }
func (f *fakeAzure) AuthState() azure.AuthState                      { return azure.AuthState{SignedIn: true} }
func (f *fakeAzure) Tenants(context.Context) ([]azure.Tenant, error) { return nil, nil }
func (f *fakeAzure) Subscriptions(context.Context, string) ([]azure.Subscription, error) {
	return nil, nil
}
func (f *fakeAzure) ResourceGroups(context.Context, string) ([]azure.ResourceGroup, error) {
	return nil, nil
}
func (f *fakeAzure) ModelResources(context.Context, string, string) ([]azure.ModelResource, error) {
	return []azure.ModelResource{f.resource}, nil
}
func (f *fakeAzure) Deployments(context.Context, string, string, string) ([]azure.Deployment, error) {
	return []azure.Deployment{{Name: "gpt-4o"}}, nil
}
func (f *fakeAzure) Models(context.Context, string, string, string) ([]azure.DeployableModel, error) {
	return []azure.DeployableModel{{Name: "gpt-4o", CodexCandidate: true}}, nil
}
func (f *fakeAzure) GetModelResource(context.Context, string, string, string) (azure.ModelResource, error) {
	return f.resource, nil
}
func (f *fakeAzure) ListKeys(context.Context, string, string, string) (azure.KeyBundle, error) {
	return f.keys, nil
}

type fakeOpenCodex struct {
	state     opencodex.State
	provider  bool
	key       string
	called    []string
	failStage string
}

func (f *fakeOpenCodex) State(context.Context) (opencodex.State, error)       { return f.state, nil }
func (f *fakeOpenCodex) Prepare(context.Context) (opencodex.State, error)     { return f.state, nil }
func (f *fakeOpenCodex) ProviderExists(context.Context, string) (bool, error) { return f.provider, nil }
func (f *fakeOpenCodex) InstallService(context.Context) error                 { return f.step("service") }
func (f *fakeOpenCodex) EnsureProvider(context.Context, string, string, string) error {
	return f.step("provider")
}
func (f *fakeOpenCodex) AddPrimaryKey(_ context.Context, _ string, key string) error {
	f.key = key
	return f.step("key")
}
func (f *fakeOpenCodex) EnsureCustomModel(context.Context, string, string) error {
	return f.step("model")
}
func (f *fakeOpenCodex) SelectModel(context.Context, string, string) error { return f.step("selected") }
func (f *fakeOpenCodex) SyncCatalog(context.Context) error                 { return f.step("catalog") }
func (f *fakeOpenCodex) TestResponse(context.Context, int, string, string) error {
	return f.step("connection")
}
func (f *fakeOpenCodex) step(name string) error {
	f.called = append(f.called, name)
	if f.failStage == name {
		return errors.New("fake failure")
	}
	return nil
}

type fakeSettings struct{ value Settings }

func (f *fakeSettings) Load() (Settings, error) {
	if f.value.ManagedProviders == nil {
		f.value.ManagedProviders = map[string]string{}
	}
	return f.value, nil
}
func (f *fakeSettings) Save(value Settings) error { f.value = value; return nil }

func newFakeService(ocx *fakeOpenCodex, settings *fakeSettings) *Service {
	return NewService(&fakeAzure{
		resource: azure.ModelResource{ID: "/subscriptions/s/resourceGroups/rg/providers/Microsoft.CognitiveServices/accounts/account", Name: "account", Endpoint: "https://account.openai.azure.com"},
		keys:     azure.KeyBundle{PrimaryKey: "primary-secret"},
	}, ocx, settings)
}

func TestSyncConvergesWithoutPersistingKey(t *testing.T) {
	ocx := &fakeOpenCodex{state: opencodex.State{NodeInfo: opencodex.NodeInfo{NodeInstalled: true, NpmInstalled: true, Compatible: true}, Installed: true, Health: opencodex.Health{Ready: true, Port: 10100}}}
	settings := &fakeSettings{}
	result := newFakeService(ocx, settings).Sync(context.Background(), SyncRequest{
		TenantID: "tenant", SubscriptionID: "subscription", ResourceGroup: "rg", ResourceName: "account", DeploymentName: "gpt-4o", ConfirmCosts: true,
	})
	if !result.OK {
		t.Fatalf("Sync failed: %#v", result)
	}
	if ocx.key != "primary-secret" {
		t.Fatalf("fake did not receive the key")
	}
	if settings.value.Selection.ProviderID != "az-account" {
		t.Fatalf("provider mapping was not saved")
	}
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), "primary-secret") {
		t.Fatalf("Sync result contains the secret")
	}
}

func TestSyncStopsAfterStageFailureAndDoesNotRollback(t *testing.T) {
	ocx := &fakeOpenCodex{state: opencodex.State{NodeInfo: opencodex.NodeInfo{NodeInstalled: true, NpmInstalled: true, Compatible: true}, Installed: true, Health: opencodex.Health{Ready: true, Port: 10100}}, failStage: "catalog"}
	settings := &fakeSettings{}
	result := newFakeService(ocx, settings).Sync(context.Background(), SyncRequest{TenantID: "tenant", SubscriptionID: "s", ResourceGroup: "rg", ResourceName: "account", DeploymentName: "gpt-4o", ConfirmCosts: true})
	if result.OK {
		t.Fatalf("Sync unexpectedly succeeded")
	}
	if len(ocx.called) != 6 {
		t.Fatalf("stages called = %v, want service through catalog", ocx.called)
	}
	if ocx.called[len(ocx.called)-1] != "catalog" {
		t.Fatalf("last stage = %v", ocx.called)
	}
	if settings.value.Selection.ProviderID != "az-account" {
		t.Fatalf("provider mapping was not saved before failure")
	}
}

func TestSyncRequiresCostConfirmation(t *testing.T) {
	ocx := &fakeOpenCodex{state: opencodex.State{NodeInfo: opencodex.NodeInfo{NodeInstalled: true, NpmInstalled: true, Compatible: true}, Installed: true}}
	result := newFakeService(ocx, &fakeSettings{}).Sync(context.Background(), SyncRequest{TenantID: "tenant", SubscriptionID: "s", ResourceGroup: "rg", ResourceName: "account", DeploymentName: "gpt-4o"})
	if result.OK || len(ocx.called) != 0 {
		t.Fatalf("Sync ran without cost confirmation: %#v", result)
	}
}
