package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/nuitsjp/foundry-codex-bridge/internal/azure"
	"github.com/nuitsjp/foundry-codex-bridge/internal/opencodex"
)

type fakeAzure struct {
	resource    azure.ModelResource
	deployments []azure.Deployment
	models      []azure.DeployableModel
	keys        azure.KeyBundle
	nilLists    bool
}

func (f *fakeAzure) Authenticate(context.Context) (azure.AuthState, error) {
	return azure.AuthState{SignedIn: true}, nil
}
func (f *fakeAzure) AuthState(context.Context) azure.AuthState {
	return azure.AuthState{CLIInstalled: true, SignedIn: true}
}
func (f *fakeAzure) Tenants(context.Context) ([]azure.Tenant, error) {
	if f.nilLists {
		return nil, nil
	}
	return []azure.Tenant{{ID: "tenant"}}, nil
}
func (f *fakeAzure) Subscriptions(context.Context, string) ([]azure.Subscription, error) {
	if f.nilLists {
		return nil, nil
	}
	return nil, nil
}
func (f *fakeAzure) ResourceGroups(context.Context, string, string) ([]azure.ResourceGroup, error) {
	if f.nilLists {
		return nil, nil
	}
	return nil, nil
}
func (f *fakeAzure) ModelResources(context.Context, string, string, string) ([]azure.ModelResource, error) {
	if f.nilLists {
		return nil, nil
	}
	return []azure.ModelResource{f.resource}, nil
}
func (f *fakeAzure) Deployments(context.Context, string, string, string, string) ([]azure.Deployment, error) {
	if f.nilLists {
		return nil, nil
	}
	return f.deployments, nil
}
func (f *fakeAzure) Models(context.Context, string, string, string, string) ([]azure.DeployableModel, error) {
	if f.nilLists {
		return nil, nil
	}
	return f.models, nil
}
func (f *fakeAzure) GetModelResource(context.Context, string, string, string, string) (azure.ModelResource, error) {
	return f.resource, nil
}
func (f *fakeAzure) ListKeys(context.Context, string, string, string, string) (azure.KeyBundle, error) {
	return f.keys, nil
}

type fakeOpenCodex struct {
	state          opencodex.State
	provider       bool
	providers      []opencodex.Provider
	combos         []opencodex.Combo
	key            string
	called         []string
	connections    []string
	failStage      string
	failConnection string
}

func (f *fakeOpenCodex) State(context.Context) (opencodex.State, error)   { return f.state, nil }
func (f *fakeOpenCodex) Prepare(context.Context) (opencodex.State, error) { return f.state, nil }
func (f *fakeOpenCodex) Providers(context.Context) ([]opencodex.Provider, error) {
	if f.providers != nil {
		return f.providers, nil
	}
	if f.provider {
		return []opencodex.Provider{{Name: "existing"}}, nil
	}
	return []opencodex.Provider{}, nil
}
func (f *fakeOpenCodex) Provider(_ context.Context, providerID string) (opencodex.Provider, error) {
	for _, provider := range f.providers {
		if provider.Name == providerID {
			return provider, nil
		}
	}
	return opencodex.Provider{}, errors.New("unknown provider")
}
func (f *fakeOpenCodex) CustomModels(context.Context) ([]opencodex.CustomModel, error) {
	return []opencodex.CustomModel{}, nil
}
func (f *fakeOpenCodex) SelectedModels(context.Context, string) ([]string, error) {
	return []string{}, nil
}
func (f *fakeOpenCodex) Combos(context.Context) ([]opencodex.Combo, error) {
	return f.combos, nil
}
func (f *fakeOpenCodex) SetDefaultProvider(_ context.Context, provider string) error {
	return f.step("default:" + provider)
}
func (f *fakeOpenCodex) RemoveProvider(_ context.Context, provider string) error {
	return f.step("remove:" + provider)
}
func (f *fakeOpenCodex) StartService(context.Context) error { return f.step("start") }
func (f *fakeOpenCodex) StopService(context.Context) error  { return f.step("stop") }
func (f *fakeOpenCodex) RepairService(context.Context) error {
	return f.step("repair")
}
func (f *fakeOpenCodex) SetPort(_ context.Context, port int) error {
	return f.step("port:" + strconv.Itoa(port))
}
func (f *fakeOpenCodex) UpdateLatest(context.Context) error { return f.step("update") }
func (f *fakeOpenCodex) SyncCatalogRestartCodex(context.Context) error {
	return f.step("restart-codex")
}
func (f *fakeOpenCodex) EnsureService(context.Context) error { return f.step("service") }
func (f *fakeOpenCodex) EnsureProvider(context.Context, string, string, string) error {
	return f.step("provider")
}
func (f *fakeOpenCodex) AddPrimaryKey(_ context.Context, _ string, key string) error {
	f.key = key
	return f.step("key")
}
func (f *fakeOpenCodex) EnsureSelectedModels(context.Context, string, []string) error {
	return f.step("selected")
}
func (f *fakeOpenCodex) EnsureCustomModels(context.Context, string, []string) error {
	return f.step("model")
}
func (f *fakeOpenCodex) ProviderExists(context.Context, string) (bool, error) { return f.provider, nil }
func (f *fakeOpenCodex) EnsureCustomModel(context.Context, string, string) error {
	return f.step("model")
}
func (f *fakeOpenCodex) SelectModel(context.Context, string, string) error { return f.step("selected") }
func (f *fakeOpenCodex) SyncCatalog(context.Context) error                 { return f.step("catalog") }
func (f *fakeOpenCodex) TestResponse(ctx context.Context, port int, provider, deployment string) error {
	return f.connectionStep(ctx, port, provider, deployment)
}
func (f *fakeOpenCodex) connectionStep(_ context.Context, _ int, _ string, deployment string) error {
	f.called = append(f.called, "connection:"+deployment)
	f.connections = append(f.connections, deployment)
	if f.failConnection == deployment {
		return errors.New("fake connection failure")
	}
	return nil
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
	normalizeSettings(&f.value)
	return f.value, nil
}
func (f *fakeSettings) Save(value Settings) error { f.value = value; return nil }

func newFakeService(ocx *fakeOpenCodex, settings *fakeSettings) *Service {
	return NewService(newFakeAzure(), ocx, settings)
}

func newFakeAzure() *fakeAzure {
	return &fakeAzure{
		resource:    azure.ModelResource{ID: "/subscriptions/s/resourceGroups/rg/providers/Microsoft.CognitiveServices/accounts/account", Name: "account", Endpoint: "https://account.openai.azure.com"},
		deployments: []azure.Deployment{{Name: "gpt-4o", ModelName: "gpt-4o", ModelFormat: "OpenAI", ModelVersion: "2024-11-20"}},
		models:      []azure.DeployableModel{{Name: "gpt-4o", Format: "OpenAI", Version: "2024-11-20", CodexCandidate: true}},
		keys:        azure.KeyBundle{PrimaryKey: "primary-secret"},
	}
}

func TestListAPIsNormalizeNilSlicesAtBridgeBoundary(t *testing.T) {
	service := NewService(&fakeAzure{nilLists: true}, &fakeOpenCodex{}, &fakeSettings{})
	checks := []struct {
		name string
		call func() (any, error)
	}{
		{name: "Tenants", call: func() (any, error) { return service.Tenants(context.Background()) }},
		{name: "Subscriptions", call: func() (any, error) { return service.Subscriptions(context.Background(), "tenant") }},
		{name: "ResourceGroups", call: func() (any, error) { return service.ResourceGroups(context.Background(), "tenant", "subscription") }},
		{name: "ModelResources", call: func() (any, error) {
			return service.ModelResources(context.Background(), "tenant", "subscription", "group")
		}},
		{name: "Deployments", call: func() (any, error) {
			return service.Deployments(context.Background(), "tenant", "subscription", "group", "resource")
		}},
		{name: "Models", call: func() (any, error) {
			return service.Models(context.Background(), "tenant", "subscription", "group", "resource")
		}},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			values, err := check.call()
			if err != nil {
				t.Fatal(err)
			}
			value := reflect.ValueOf(values)
			if value.Kind() != reflect.Slice || value.IsNil() {
				t.Fatalf("result = %#v, want a non-nil slice", values)
			}
			encoded, err := json.Marshal(values)
			if err != nil {
				t.Fatal(err)
			}
			if string(encoded) != "[]" {
				t.Fatalf("JSON result = %s, want []", encoded)
			}
		})
	}
}

func TestSyncConvergesWithoutPersistingKey(t *testing.T) {
	ocx := &fakeOpenCodex{state: opencodex.State{NodeInfo: opencodex.NodeInfo{NodeInstalled: true, NpmInstalled: true, Compatible: true}, Installed: true, Health: opencodex.Health{Ready: true, Port: 10100}}}
	settings := &fakeSettings{}
	result := newFakeService(ocx, settings).Sync(context.Background(), SyncRequest{
		TenantID: "tenant", SubscriptionID: "subscription", ResourceGroup: "rg", ResourceName: "account", DeploymentNames: []string{"gpt-4o"}, DefaultDeploymentName: "gpt-4o", ProviderID: " az-account ", ConfirmCosts: true,
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
	wantCalls := []string{"service", "provider", "key", "selected", "model", "catalog", "connection:gpt-4o"}
	if !reflect.DeepEqual(ocx.called, wantCalls) {
		t.Fatalf("stages called = %v, want %v", ocx.called, wantCalls)
	}
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), "primary-secret") {
		t.Fatalf("Sync result contains the secret")
	}
}

func TestSyncStopsAfterStageFailureAndDoesNotRollback(t *testing.T) {
	ocx := &fakeOpenCodex{state: opencodex.State{NodeInfo: opencodex.NodeInfo{NodeInstalled: true, NpmInstalled: true, Compatible: true}, Installed: true, Health: opencodex.Health{Ready: true, Port: 10100}}, failStage: "catalog"}
	settings := &fakeSettings{}
	result := newFakeService(ocx, settings).Sync(context.Background(), SyncRequest{TenantID: "tenant", SubscriptionID: "s", ResourceGroup: "rg", ResourceName: "account", DeploymentNames: []string{"gpt-4o"}, DefaultDeploymentName: "gpt-4o", ConfirmCosts: true})
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
	statuses := map[string]string{}
	for _, stage := range result.Stages {
		statuses[stage.Name] = stage.Status
	}
	if statuses["catalog"] != "failed" || statuses["connection"] != "pending" {
		t.Fatalf("stage statuses = %#v", statuses)
	}
}

func TestSyncRequiresCostConfirmation(t *testing.T) {
	ocx := &fakeOpenCodex{state: opencodex.State{NodeInfo: opencodex.NodeInfo{NodeInstalled: true, NpmInstalled: true, Compatible: true}, Installed: true}}
	result := newFakeService(ocx, &fakeSettings{}).Sync(context.Background(), SyncRequest{TenantID: "tenant", SubscriptionID: "s", ResourceGroup: "rg", ResourceName: "account", DeploymentNames: []string{"gpt-4o"}, DefaultDeploymentName: "gpt-4o"})
	if result.OK || len(ocx.called) != 0 {
		t.Fatalf("Sync ran without cost confirmation: %#v", result)
	}
}

func TestSyncRejectsDeploymentOutsideCodexCandidatesBeforeOpenCodexChanges(t *testing.T) {
	ocx := &fakeOpenCodex{state: opencodex.State{NodeInfo: opencodex.NodeInfo{NodeInstalled: true, NpmInstalled: true, Compatible: true}, Installed: true}}
	azureClient := newFakeAzure()
	azureClient.models[0].CodexCandidate = false
	service := NewService(azureClient, ocx, &fakeSettings{})

	result := service.Sync(context.Background(), SyncRequest{TenantID: "tenant", SubscriptionID: "s", ResourceGroup: "rg", ResourceName: "account", DeploymentNames: []string{"gpt-4o"}, DefaultDeploymentName: "gpt-4o", ConfirmCosts: true})

	if result.OK || len(ocx.called) != 0 {
		t.Fatalf("Sync changed opencodex for an unsupported deployment: %#v, calls=%v", result, ocx.called)
	}
	if len(result.Stages) != 1 || result.Stages[0].Name != "azure-deployment" {
		t.Fatalf("unexpected failure stage: %#v", result.Stages)
	}
}

func TestSyncSupportsMultipleDeploymentsInAzureOrder(t *testing.T) {
	azureClient := newFakeAzure()
	azureClient.deployments = []azure.Deployment{
		{Name: "second", ModelName: "second", ModelFormat: "OpenAI"},
		{Name: "first", ModelName: "first", ModelFormat: "OpenAI"},
	}
	azureClient.models = []azure.DeployableModel{
		{Name: "first", Format: "OpenAI", CodexCandidate: true},
		{Name: "second", Format: "OpenAI", CodexCandidate: true},
	}
	ocx := &fakeOpenCodex{state: opencodex.State{NodeInfo: opencodex.NodeInfo{NodeInstalled: true, NpmInstalled: true, Compatible: true}, Installed: true, Health: opencodex.Health{Ready: true, Port: 10100}}}
	result := NewService(azureClient, ocx, &fakeSettings{}).Sync(context.Background(), SyncRequest{
		TenantID: "tenant", SubscriptionID: "s", ResourceGroup: "rg", ResourceName: "account",
		DeploymentNames: []string{" first ", "second", "first"}, DefaultDeploymentName: "second", ConfirmCosts: true,
	})
	if !result.OK || !reflect.DeepEqual(ocx.connections, []string{"second", "first"}) {
		t.Fatalf("multi-deployment sync = %#v, connections=%v", result, ocx.connections)
	}
	if got := result.Connections[0].Deployment; got != "second" {
		t.Fatalf("first connection deployment = %q", got)
	}
}

func TestSyncRejectsDefaultOutsideDeploymentSetBeforeOpenCodexChanges(t *testing.T) {
	ocx := &fakeOpenCodex{state: opencodex.State{NodeInfo: opencodex.NodeInfo{NodeInstalled: true, NpmInstalled: true, Compatible: true}, Installed: true}}
	result := newFakeService(ocx, &fakeSettings{}).Sync(context.Background(), SyncRequest{
		TenantID: "tenant", SubscriptionID: "s", ResourceGroup: "rg", ResourceName: "account",
		DeploymentNames: []string{"gpt-4o"}, DefaultDeploymentName: "other", ConfirmCosts: true,
	})
	if result.OK || len(ocx.called) != 0 || len(result.Stages) != 1 || result.Stages[0].Name != "validate" {
		t.Fatalf("default validation = %#v, calls=%v", result, ocx.called)
	}
}

func TestSyncTestsEveryConnectionAfterOneFailure(t *testing.T) {
	azureClient := newFakeAzure()
	azureClient.deployments = []azure.Deployment{
		{Name: "first", ModelName: "first", ModelFormat: "OpenAI"},
		{Name: "second", ModelName: "second", ModelFormat: "OpenAI"},
	}
	azureClient.models = []azure.DeployableModel{{Name: "first", Format: "OpenAI", CodexCandidate: true}, {Name: "second", Format: "OpenAI", CodexCandidate: true}}
	ocx := &fakeOpenCodex{failConnection: "first", state: opencodex.State{NodeInfo: opencodex.NodeInfo{NodeInstalled: true, NpmInstalled: true, Compatible: true}, Installed: true, Health: opencodex.Health{Ready: true, Port: 10100}}}
	result := NewService(azureClient, ocx, &fakeSettings{}).Sync(context.Background(), SyncRequest{
		TenantID: "tenant", SubscriptionID: "s", ResourceGroup: "rg", ResourceName: "account",
		DeploymentNames: []string{"first", "second"}, DefaultDeploymentName: "first", ConfirmCosts: true,
	})
	if result.OK || !reflect.DeepEqual(ocx.connections, []string{"first", "second"}) || len(result.Connections) != 2 {
		t.Fatalf("connection continuation = %#v, connections=%v", result, ocx.connections)
	}
	if result.Connections[0].Status != "failed" || result.Connections[1].Status != "succeeded" {
		t.Fatalf("connection results = %#v", result.Connections)
	}
}

func TestSyncPreservesOtherManagedProviderMappings(t *testing.T) {
	settings := &fakeSettings{value: Settings{ManagedProviders: map[string]ManagedProvider{
		"other-resource": {ResourceID: "other-resource", ProviderID: "az-other", DeploymentNames: []string{"other"}},
	}}}
	ocx := &fakeOpenCodex{state: opencodex.State{NodeInfo: opencodex.NodeInfo{NodeInstalled: true, NpmInstalled: true, Compatible: true}, Installed: true, Health: opencodex.Health{Ready: true, Port: 10100}}}
	result := newFakeService(ocx, settings).Sync(context.Background(), SyncRequest{
		TenantID: "tenant", SubscriptionID: "s", ResourceGroup: "rg", ResourceName: "account",
		DeploymentNames: []string{"gpt-4o"}, DefaultDeploymentName: "gpt-4o", ConfirmCosts: true,
	})
	if !result.OK {
		t.Fatalf("Sync failed: %#v", result)
	}
	if settings.value.ManagedProviders["other-resource"].ProviderID != "az-other" {
		t.Fatalf("other managed provider mapping was lost: %#v", settings.value.ManagedProviders)
	}
}
