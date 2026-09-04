package bridge

import (
	"context"
	"reflect"
	"testing"

	"github.com/nuitsjp/foundry-codex-bridge/internal/opencodex"
)

func managedSettings() *fakeSettings {
	resourceID := "/subscriptions/s/resourceGroups/rg/providers/Microsoft.CognitiveServices/accounts/account"
	return &fakeSettings{value: Settings{ManagedProviders: map[string]ManagedProvider{
		resourceID: {ResourceID: resourceID, ProviderID: "az-account", ResourceName: "account", SubscriptionID: "s", Location: "eastus"},
	}, Selection: Selection{ResourceID: resourceID, ProviderID: "az-account"}}}
}

func TestPreviewDisconnectBlocksComboDependency(t *testing.T) {
	settings := managedSettings()
	ocx := &fakeOpenCodex{
		providers: []opencodex.Provider{{Name: "openai", IsDefault: true}, {Name: "az-account"}},
		combos:    []opencodex.Combo{{ID: "fallback", Targets: []opencodex.ComboTarget{{Provider: "az-account", Model: "gpt"}}}},
	}
	preview := NewService(newFakeAzure(), ocx, settings).PreviewDisconnect(context.Background(), settings.value.Selection.ResourceID)
	if preview.OK || !reflect.DeepEqual(preview.DependentCombos, []string{"fallback"}) {
		t.Fatalf("PreviewDisconnect() = %#v", preview)
	}
}

func TestPreviewSyncIsReadOnlyAndShowsPlannedProviderCreation(t *testing.T) {
	ocx := &fakeOpenCodex{state: opencodex.State{Installed: true, Health: opencodex.Health{Ready: true}}}
	service := NewService(newFakeAzure(), ocx, &fakeSettings{})
	preview := service.PreviewSync(context.Background(), SyncRequest{
		TenantID: "tenant", SubscriptionID: "s", ResourceGroup: "rg", ResourceName: "account",
		DeploymentNames: []string{"gpt-4o"}, DefaultDeploymentName: "gpt-4o",
	})
	if !preview.OK || preview.ProviderID != "az-account" {
		t.Fatalf("PreviewSync() = %#v", preview)
	}
	if len(ocx.called) != 0 {
		t.Fatalf("PreviewSync mutated opencodex: %v", ocx.called)
	}
	foundCreate := false
	for _, change := range preview.Changes {
		if change.Area == "provider" && change.Action == "create" {
			foundCreate = true
		}
	}
	if !foundCreate {
		t.Fatalf("PreviewSync changes = %#v", preview.Changes)
	}
}

func TestPreviewSyncShowsLiveModelsDriftForManagedProvider(t *testing.T) {
	settings := managedSettings()
	ocx := &fakeOpenCodex{
		state:     opencodex.State{Installed: true, Health: opencodex.Health{Ready: true}},
		providers: []opencodex.Provider{{Name: "az-account", Adapter: "azure-openai", BaseURL: "https://account.openai.azure.com/openai", DefaultModel: "gpt-4o", LiveModels: true}},
	}
	preview := NewService(newFakeAzure(), ocx, settings).PreviewSync(context.Background(), SyncRequest{
		TenantID: "tenant", SubscriptionID: "s", ResourceGroup: "rg", ResourceName: "account",
		DeploymentNames: []string{"gpt-4o"}, DefaultDeploymentName: "gpt-4o", ProviderID: "az-account",
	})
	if !preview.OK {
		t.Fatalf("PreviewSync() = %#v", preview)
	}
	found := false
	for _, change := range preview.Changes {
		if change.Area == "provider" && change.Action == "update" {
			found = true
		}
	}
	if !found {
		t.Fatalf("provider drift missing: %#v", preview.Changes)
	}
}

func TestDisconnectMovesDefaultThenRemovesMappingAndSyncs(t *testing.T) {
	settings := managedSettings()
	ocx := &fakeOpenCodex{providers: []opencodex.Provider{{Name: "az-account", IsDefault: true}, {Name: "openai"}}}
	service := NewService(newFakeAzure(), ocx, settings)
	result := service.Disconnect(context.Background(), DisconnectRequest{ResourceID: settings.value.Selection.ResourceID, ReplacementProviderID: "openai"})
	if !result.OK {
		t.Fatalf("Disconnect() = %#v", result)
	}
	want := []string{"default:openai", "remove:az-account", "catalog"}
	if !reflect.DeepEqual(ocx.called, want) {
		t.Fatalf("calls = %v, want %v", ocx.called, want)
	}
	if len(settings.value.ManagedProviders) != 0 || settings.value.Selection.ResourceID != "" {
		t.Fatalf("settings were not cleared: %#v", settings.value)
	}
}

func TestDisconnectKeepsMappingWhenProviderRemovalFails(t *testing.T) {
	settings := managedSettings()
	ocx := &fakeOpenCodex{providers: []opencodex.Provider{{Name: "openai", IsDefault: true}, {Name: "az-account"}}, failStage: "remove:az-account"}
	result := NewService(newFakeAzure(), ocx, settings).Disconnect(context.Background(), DisconnectRequest{ResourceID: settings.value.Selection.ResourceID})
	if result.OK || len(settings.value.ManagedProviders) != 1 {
		t.Fatalf("Disconnect() = %#v, settings=%#v", result, settings.value)
	}
}

func TestChangeOpenCodexPortRunsRequiredSequence(t *testing.T) {
	ocx := &fakeOpenCodex{}
	result := NewService(newFakeAzure(), ocx, &fakeSettings{}).ChangeOpenCodexPort(context.Background(), 10101)
	if !result.OK {
		t.Fatalf("ChangeOpenCodexPort() = %#v", result)
	}
	want := []string{"port:10101", "stop", "start", "catalog"}
	if !reflect.DeepEqual(ocx.called, want) {
		t.Fatalf("calls = %v, want %v", ocx.called, want)
	}
}
