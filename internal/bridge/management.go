package bridge

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
)

func (s *Service) ManagedProviders() ([]ManagedProvider, error) {
	settings, err := s.settings.Load()
	if err != nil {
		return nil, errors.New("Bridge settings could not be read")
	}
	providers := make([]ManagedProvider, 0, len(settings.ManagedProviders))
	for _, provider := range settings.ManagedProviders {
		providers = append(providers, provider)
	}
	sort.Slice(providers, func(i, j int) bool { return providers[i].ProviderID < providers[j].ProviderID })
	return providers, nil
}

func (s *Service) PreviewSync(ctx context.Context, request SyncRequest) SyncPreview {
	ctx = s.context(ctx)
	request.ConfirmCosts = true
	if err := validateSyncRequest(request); err != nil {
		return SyncPreview{Message: err.Error(), Changes: []SyncChange{}}
	}
	resource, err := s.azure.GetModelResource(ctx, request.TenantID, request.SubscriptionID, request.ResourceGroup, request.ResourceName)
	if err != nil {
		return SyncPreview{Message: safeErrorMessage(err), Changes: []SyncChange{}}
	}
	if resource.DisableLocalAuth || strings.TrimSpace(resource.Endpoint) == "" {
		return SyncPreview{Message: "the selected Azure Model Resource cannot use API-key Sync", Changes: []SyncChange{}}
	}
	desired := normalizeDeploymentInput(request.DeploymentNames, request.DefaultDeploymentName)
	deployments, err := s.azure.Deployments(ctx, request.TenantID, request.SubscriptionID, request.ResourceGroup, request.ResourceName)
	if err != nil {
		return SyncPreview{Message: safeErrorMessage(err), Changes: []SyncChange{}}
	}
	models, err := s.azure.Models(ctx, request.TenantID, request.SubscriptionID, request.ResourceGroup, request.ResourceName)
	if err != nil {
		return SyncPreview{Message: safeErrorMessage(err), Changes: []SyncChange{}}
	}
	available := make(map[string]Deployment, len(deployments))
	for _, deployment := range deployments {
		available[deployment.Name] = deployment
	}
	for _, name := range desired {
		deployment, ok := available[name]
		if !ok {
			return SyncPreview{Message: "one or more selected deployments do not exist in the Azure Model Resource", Changes: []SyncChange{}}
		}
		if !deploymentIsCodexCandidate(deployment, models) {
			return SyncPreview{Message: "one or more selected deployments are not supported by the opencodex Codex route", Changes: []SyncChange{}}
		}
	}
	settings, err := s.settings.Load()
	if err != nil {
		return SyncPreview{Message: "Bridge settings could not be read", Changes: []SyncChange{}}
	}
	normalizeSettings(&settings)
	managed := settings.ManagedProviders[resource.ID]
	providerID := strings.TrimSpace(request.ProviderID)
	if providerID == "" {
		providerID = managed.ProviderID
	}
	if providerID == "" {
		providerID = defaultProviderID(resource.Name)
	}
	if !validProviderID(providerID) {
		return SyncPreview{ProviderID: providerID, Message: "Provider ID may contain only letters, numbers, dot, underscore, and hyphen", Changes: []SyncChange{}}
	}
	if managed.ProviderID != "" && managed.ProviderID != providerID {
		return SyncPreview{ProviderID: providerID, Message: "a different Provider ID is already managed for this Azure Model Resource", Changes: []SyncChange{}}
	}
	preview := SyncPreview{OK: true, ProviderID: providerID, Changes: []SyncChange{}}
	state, _ := s.opencodex.State(ctx)
	if !state.Health.Ready {
		preview.Changes = append(preview.Changes, SyncChange{Area: "service", Action: "repair", Details: "Ensure the opencodex background service is ready"})
	}
	providers, err := s.opencodex.Providers(ctx)
	if err != nil {
		return SyncPreview{ProviderID: providerID, Message: safeErrorMessage(err), Changes: []SyncChange{}}
	}
	baseURL := openAIBaseURL(resource.Endpoint)
	found := false
	for _, provider := range providers {
		if provider.Name != providerID {
			continue
		}
		found = true
		if managed.ProviderID == "" {
			return SyncPreview{ProviderID: providerID, Message: "Provider ID is already used by an unmanaged opencodex Provider", Changes: []SyncChange{}}
		}
		current, err := s.opencodex.Provider(ctx, providerID)
		if err != nil {
			return SyncPreview{ProviderID: providerID, Message: safeErrorMessage(err), Changes: []SyncChange{}}
		}
		if current.Adapter != "azure-openai" || current.BaseURL != baseURL || current.DefaultModel != strings.TrimSpace(request.DefaultDeploymentName) || current.LiveModels {
			preview.Changes = append(preview.Changes, SyncChange{Area: "provider", Action: "update", Details: "Reconcile adapter, endpoint, default model, and live models"})
		}
	}
	if !found {
		preview.Changes = append(preview.Changes, SyncChange{Area: "provider", Action: "create", Details: "Create Bridge-managed Provider " + providerID})
		preview.Changes = append(preview.Changes,
			SyncChange{Area: "selected-models", Action: "update", Details: strings.Join(desired, ", ")},
			SyncChange{Area: "custom-models", Action: "update", Details: strings.Join(desired, ", ")},
		)
	}
	if found {
		selected, err := s.opencodex.SelectedModels(ctx, providerID)
		if err != nil {
			return SyncPreview{ProviderID: providerID, Message: safeErrorMessage(err), Changes: []SyncChange{}}
		}
		if !sameNames(selected, desired) {
			preview.Changes = append(preview.Changes, SyncChange{Area: "selected-models", Action: "update", Details: strings.Join(desired, ", ")})
		}
		custom, err := s.opencodex.CustomModels(ctx)
		if err != nil {
			return SyncPreview{ProviderID: providerID, Message: safeErrorMessage(err), Changes: []SyncChange{}}
		}
		current := make([]string, 0)
		for _, model := range custom {
			if model.Provider == providerID {
				current = append(current, model.ModelID)
			}
		}
		if !sameNames(current, desired) {
			preview.Changes = append(preview.Changes, SyncChange{Area: "custom-models", Action: "update", Details: strings.Join(desired, ", ")})
		}
	}
	preview.Changes = append(preview.Changes,
		SyncChange{Area: "credential", Action: "refresh", Details: "Register the current Azure PrimaryKey through stdin"},
		SyncChange{Area: "catalog", Action: "sync", Details: "Refresh the Codex model catalog"},
		SyncChange{Area: "connection", Action: "test", Details: "Send one Responses request per selected deployment"},
	)
	return preview
}

func sameNames(left, right []string) bool {
	left = normalizeNames(left)
	right = normalizeNames(right)
	sort.Strings(left)
	sort.Strings(right)
	return strings.Join(left, "\x00") == strings.Join(right, "\x00")
}

func (s *Service) PreviewDisconnect(ctx context.Context, resourceID string) DisconnectPreview {
	ctx = s.context(ctx)
	settings, err := s.settings.Load()
	if err != nil {
		return DisconnectPreview{Message: "Bridge settings could not be read", DependentCombos: []string{}, ReplacementProviders: []string{}}
	}
	managed, ok := settings.ManagedProviders[strings.TrimSpace(resourceID)]
	if !ok {
		return DisconnectPreview{Message: "the selected Bridge-managed Provider does not exist", DependentCombos: []string{}, ReplacementProviders: []string{}}
	}
	preview := DisconnectPreview{OK: true, Managed: managed, DependentCombos: []string{}, ReplacementProviders: []string{}}
	providers, err := s.opencodex.Providers(ctx)
	if err != nil {
		preview.OK = false
		preview.Message = safeErrorMessage(err)
		return preview
	}
	for _, provider := range providers {
		if provider.Name == managed.ProviderID {
			preview.IsDefault = provider.IsDefault
			continue
		}
		preview.ReplacementProviders = append(preview.ReplacementProviders, provider.Name)
	}
	combos, err := s.opencodex.Combos(ctx)
	if err != nil {
		preview.OK = false
		preview.Message = safeErrorMessage(err)
		return preview
	}
	for _, combo := range combos {
		for _, target := range combo.Targets {
			if target.Provider == managed.ProviderID {
				preview.DependentCombos = append(preview.DependentCombos, combo.ID)
				break
			}
		}
	}
	if len(preview.DependentCombos) != 0 {
		preview.OK = false
		preview.Message = "the Provider is referenced by one or more Combos"
	} else if preview.IsDefault && len(preview.ReplacementProviders) == 0 {
		preview.OK = false
		preview.Message = "the default Provider cannot be removed without a replacement"
	}
	return preview
}

func (s *Service) Disconnect(ctx context.Context, request DisconnectRequest) ActionResult {
	ctx = s.context(ctx)
	resourceID := strings.TrimSpace(request.ResourceID)
	result := ActionResult{Stages: []SyncStage{}}
	if !s.mu.TryLock() {
		return failAction(result, "disconnect", "another modifying operation is already running")
	}
	defer s.mu.Unlock()
	preview := s.PreviewDisconnect(ctx, resourceID)
	if !preview.OK {
		return failAction(result, "validate", preview.Message)
	}
	if preview.IsDefault {
		replacement := strings.TrimSpace(request.ReplacementProviderID)
		if replacement == "" || replacement == preview.Managed.ProviderID || !containsName(preview.ReplacementProviders, replacement) {
			return failAction(result, "validate", "select a valid replacement for the default Provider")
		}
		if err := s.runActionStage(&result, "default", "Set replacement default Provider", func() error { return s.opencodex.SetDefaultProvider(ctx, replacement) }); err != nil {
			return result
		}
	}
	if err := s.runActionStage(&result, "provider", "Remove Bridge-managed Provider", func() error { return s.opencodex.RemoveProvider(ctx, preview.Managed.ProviderID) }); err != nil {
		return result
	}
	settings, err := s.settings.Load()
	if err != nil {
		return failAction(result, "settings", "Bridge settings could not be read")
	}
	delete(settings.ManagedProviders, resourceID)
	if settings.Selection.ResourceID == resourceID {
		settings.Selection = Selection{}
	}
	if err := settingsSaveStage(s, &result, settings); err != nil {
		return result
	}
	if err := s.runActionStage(&result, "catalog", "Sync Codex Catalog", func() error { return s.opencodex.SyncCatalog(ctx) }); err != nil {
		return result
	}
	result.OK = true
	return result
}

func settingsSaveStage(s *Service, result *ActionResult, settings Settings) error {
	return s.runActionStage(result, "settings", "Remove Bridge mapping", func() error { return s.settings.Save(settings) })
}

func containsName(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func (s *Service) StartOpenCodex(ctx context.Context) ActionResult {
	return s.runOpenCodexAction(ctx, "service", "Start opencodex service", s.opencodex.StartService)
}

func (s *Service) StopOpenCodex(ctx context.Context) ActionResult {
	return s.runOpenCodexAction(ctx, "service", "Stop opencodex service", s.opencodex.StopService)
}

func (s *Service) RepairOpenCodex(ctx context.Context) ActionResult {
	return s.runOpenCodexAction(ctx, "service", "Repair opencodex service", s.opencodex.RepairService)
}

func (s *Service) UpdateOpenCodex(ctx context.Context) ActionResult {
	return s.runOpenCodexAction(ctx, "update", "Update opencodex to latest", s.opencodex.UpdateLatest)
}

func (s *Service) RestartCodexCatalog(ctx context.Context) ActionResult {
	return s.runOpenCodexAction(ctx, "catalog", "Sync catalog and restart Codex", s.opencodex.SyncCatalogRestartCodex)
}

func (s *Service) ChangeOpenCodexPort(ctx context.Context, port int) ActionResult {
	ctx = s.context(ctx)
	result := ActionResult{Stages: []SyncStage{}}
	if !s.mu.TryLock() {
		return failAction(result, "port", "another modifying operation is already running")
	}
	defer s.mu.Unlock()
	steps := []struct {
		name, message string
		run           func() error
	}{
		{"port", "Set opencodex port to " + strconv.Itoa(port), func() error { return s.opencodex.SetPort(ctx, port) }},
		{"stop", "Stop opencodex service", func() error { return s.opencodex.StopService(ctx) }},
		{"start", "Start opencodex service", func() error { return s.opencodex.StartService(ctx) }},
		{"catalog", "Sync Codex Catalog", func() error { return s.opencodex.SyncCatalog(ctx) }},
	}
	for _, step := range steps {
		if err := s.runActionStage(&result, step.name, step.message, step.run); err != nil {
			return result
		}
	}
	result.OK = true
	return result
}

func (s *Service) runOpenCodexAction(ctx context.Context, name, message string, action func(context.Context) error) ActionResult {
	ctx = s.context(ctx)
	result := ActionResult{Stages: []SyncStage{}}
	if !s.mu.TryLock() {
		return failAction(result, name, "another modifying operation is already running")
	}
	defer s.mu.Unlock()
	if err := s.runActionStage(&result, name, message, func() error { return action(ctx) }); err != nil {
		return result
	}
	result.OK = true
	return result
}

func (s *Service) runActionStage(result *ActionResult, name, message string, action func() error) error {
	if err := action(); err != nil {
		result.Stages = append(result.Stages, SyncStage{Name: name, Status: "failed", Message: safeErrorMessage(err)})
		return err
	}
	result.Stages = append(result.Stages, SyncStage{Name: name, Status: "succeeded", Message: message})
	return nil
}

func failAction(result ActionResult, name, message string) ActionResult {
	result.Stages = append(result.Stages, SyncStage{Name: name, Status: "failed", Message: message})
	return result
}
