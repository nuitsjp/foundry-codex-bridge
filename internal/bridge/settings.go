package bridge

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type FileSettingsStore struct {
	path string
}

func NewFileSettingsStore(dataDir string) *FileSettingsStore {
	return &FileSettingsStore{path: filepath.Join(dataDir, "settings.json")}
}

func (s *FileSettingsStore) Load() (Settings, error) {
	result := Settings{ManagedProviders: map[string]string{}}
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return Settings{}, err
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return Settings{}, err
	}
	if result.ManagedProviders == nil {
		result.ManagedProviders = map[string]string{}
	}
	return result, nil
}

func (s *FileSettingsStore) Save(settings Settings) error {
	if settings.ManagedProviders == nil {
		settings.ManagedProviders = map[string]string{}
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o600)
}
