package bridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSettingsLoadMigratesLegacyManagedProvidersAndDeployment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	legacy := `{"selection":{"tenantId":"tenant","deploymentName":"gpt-4o"},"managedProviders":{"resource-1":"az-account"}}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &FileSettingsStore{path: path}
	settings, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if settings.Selection.DefaultDeploymentName != "gpt-4o" || len(settings.Selection.DeploymentNames) != 1 {
		t.Fatalf("selection migration = %#v", settings.Selection)
	}
	managed := settings.ManagedProviders["resource-1"]
	if managed.ProviderID != "az-account" || managed.ResourceID != "resource-1" {
		t.Fatalf("managed provider migration = %#v", managed)
	}
}

func TestSettingsSaveWritesOnlyNewShapeAndNoSecret(t *testing.T) {
	dir := t.TempDir()
	store := NewFileSettingsStore(dir)
	settings := Settings{
		Selection:        Selection{DeploymentNames: []string{"gpt-4o"}, DefaultDeploymentName: "gpt-4o"},
		ManagedProviders: map[string]ManagedProvider{"resource-1": {ResourceID: "resource-1", ProviderID: "az-account", DeploymentNames: []string{"gpt-4o"}}},
	}
	if err := store.Save(settings); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, `"deploymentName":`) || strings.Contains(text, "primary-secret") || strings.Contains(text, "apiKey") {
		t.Fatalf("saved settings contain legacy field or secret: %s", text)
	}
}
