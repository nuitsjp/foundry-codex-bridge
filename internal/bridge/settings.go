package bridge

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type FileSettingsStore struct {
	path string
}

func NewFileSettingsStore(dataDir string) *FileSettingsStore {
	return &FileSettingsStore{path: filepath.Join(dataDir, "settings.json")}
}

func (s *FileSettingsStore) Load() (Settings, error) {
	result := Settings{ManagedProviders: map[string]ManagedProvider{}}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return Settings{}, err
	}
	if err := unmarshalSettings(data, &result); err != nil {
		return Settings{}, err
	}
	normalizeSettings(&result)
	return result, nil
}

func (s *FileSettingsStore) Save(settings Settings) error {
	normalizeSettings(&settings)
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o600)
}

func unmarshalSettings(data []byte, result *Settings) error {
	type settingsWire struct {
		Selection        json.RawMessage `json:"selection"`
		ManagedProviders json.RawMessage `json:"managedProviders"`
	}
	var wire settingsWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if len(wire.Selection) != 0 && string(wire.Selection) != "null" {
		if err := json.Unmarshal(wire.Selection, &result.Selection); err != nil {
			return err
		}
		var legacy struct {
			DeploymentName string `json:"deploymentName"`
		}
		if err := json.Unmarshal(wire.Selection, &legacy); err != nil {
			return err
		}
		result.Selection.DeploymentName = legacy.DeploymentName
		if len(result.Selection.DeploymentNames) == 0 && legacy.DeploymentName != "" {
			result.Selection.DeploymentNames = []string{legacy.DeploymentName}
			result.Selection.DefaultDeploymentName = legacy.DeploymentName
		}
	}
	if len(wire.ManagedProviders) != 0 && string(wire.ManagedProviders) != "null" {
		if err := json.Unmarshal(wire.ManagedProviders, &result.ManagedProviders); err != nil {
			var legacy map[string]string
			if legacyErr := json.Unmarshal(wire.ManagedProviders, &legacy); legacyErr != nil {
				return err
			}
			result.ManagedProviders = make(map[string]ManagedProvider, len(legacy))
			for resourceID, providerID := range legacy {
				result.ManagedProviders[resourceID] = ManagedProvider{ResourceID: resourceID, ProviderID: providerID}
			}
		}
	}
	return nil
}

func normalizeSettings(settings *Settings) {
	if settings.ManagedProviders == nil {
		settings.ManagedProviders = map[string]ManagedProvider{}
	}
	settings.Selection.DeploymentNames = normalizeNames(settings.Selection.DeploymentNames)
	for resourceID, provider := range settings.ManagedProviders {
		if provider.ResourceID == "" {
			provider.ResourceID = resourceID
		}
		provider.DeploymentNames = normalizeNames(provider.DeploymentNames)
		settings.ManagedProviders[resourceID] = provider
	}
}

func normalizeNames(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
