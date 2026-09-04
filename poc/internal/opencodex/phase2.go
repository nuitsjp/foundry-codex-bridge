package opencodex

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

func (m *Manager) Providers(ctx context.Context) ([]Provider, error) {
	result, err := m.runner.Run(ctx, "", "provider", "list", "--json")
	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, commandFailure("ocx provider list", result)
	}
	var payload struct {
		Configured []Provider `json:"configured"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		return nil, errors.New("ocx provider list returned invalid JSON")
	}
	if payload.Configured == nil {
		payload.Configured = []Provider{}
	}
	return payload.Configured, nil
}

func (m *Manager) Provider(ctx context.Context, providerID string) (Provider, error) {
	return m.providerDetails(ctx, providerID)
}

func (m *Manager) CustomModels(ctx context.Context) ([]CustomModel, error) {
	result, err := m.runner.Run(ctx, "", "models", "list-custom", "--json")
	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, commandFailure("ocx models list-custom", result)
	}
	var models []CustomModel
	if err := json.Unmarshal([]byte(result.Stdout), &models); err != nil {
		return nil, errors.New("ocx models list-custom returned invalid JSON")
	}
	if models == nil {
		models = []CustomModel{}
	}
	return models, nil
}

func (m *Manager) SelectedModels(ctx context.Context, providerID string) ([]string, error) {
	result, err := m.runner.Run(ctx, "", "models", "selected", providerID, "--json")
	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, commandFailure("ocx models selected", result)
	}
	var payload struct {
		Selected []string `json:"selected"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		return nil, errors.New("ocx models selected returned invalid JSON")
	}
	if payload.Selected == nil {
		payload.Selected = []string{}
	}
	return payload.Selected, nil
}

func (m *Manager) EnsureSelectedModels(ctx context.Context, providerID string, desired []string) error {
	current, err := m.SelectedModels(ctx, providerID)
	if err != nil {
		return err
	}
	want := sortedUnique(desired)
	if equalStringSets(current, want) {
		return nil
	}

	args := []string{"models", "selected", providerID}
	if len(want) == 0 {
		args = append(args, "--clear", "--json")
	} else {
		args = append(args, "--set", strings.Join(want, ","), "--json")
	}
	result, err := m.runner.Run(ctx, "", args...)
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return commandFailure("ocx models selected", result)
	}
	return nil
}

func (m *Manager) EnsureCustomModels(ctx context.Context, providerID string, desired []string) error {
	current, err := m.CustomModels(ctx)
	if err != nil {
		return err
	}
	want := sortedUnique(desired)
	currentForProvider := make([]string, 0, len(current))
	for _, model := range current {
		if strings.TrimSpace(model.Provider) == strings.TrimSpace(providerID) {
			currentForProvider = append(currentForProvider, model.ModelID)
		}
	}
	currentForProvider = sortedUnique(currentForProvider)

	currentSet := stringSet(currentForProvider)
	wantSet := stringSet(want)
	for _, model := range want {
		if _, ok := currentSet[model]; ok {
			continue
		}
		result, err := m.runner.Run(ctx, "", "models", "add", providerID, model, "--modalities", "text")
		if err != nil {
			return err
		}
		if result.Code != 0 {
			return commandFailure("ocx models add", result)
		}
	}
	for _, model := range currentForProvider {
		if _, ok := wantSet[model]; ok {
			continue
		}
		result, err := m.runner.Run(ctx, "", "models", "remove", providerID+"/"+model, "--yes")
		if err != nil {
			return err
		}
		if result.Code != 0 {
			return commandFailure("ocx models remove", result)
		}
	}
	return nil
}

func sortedUnique(values []string) []string {
	set := stringSet(nil)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func equalStringSets(left, right []string) bool {
	leftSet := stringSet(sortedUnique(left))
	rightSet := stringSet(sortedUnique(right))
	if len(leftSet) != len(rightSet) {
		return false
	}
	for value := range leftSet {
		if _, ok := rightSet[value]; !ok {
			return false
		}
	}
	return true
}

var _ ManagerAPI = (*Manager)(nil)
