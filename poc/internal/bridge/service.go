package bridge

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nuitsjp/foundry-codex-bridge/internal/azure"
	"github.com/nuitsjp/foundry-codex-bridge/internal/opencodex"
)

func (s *Service) Snapshot(ctx context.Context) Snapshot {
	ctx = s.context(ctx)
	settings, err := s.settings.Load()
	if err != nil {
		settings = Settings{ManagedProviders: map[string]ManagedProvider{}}
	}
	state, _ := s.opencodex.State(ctx)
	auth := s.azure.AuthState(ctx)
	return Snapshot{
		Auth:             auth,
		Selection:        settings.Selection,
		OpenCodex:        state,
		AzureClientReady: auth.SignedIn,
	}
}

func (s *Service) SignIn(ctx context.Context) (AuthState, error) {
	return s.azure.Authenticate(s.context(ctx))
}

func (s *Service) Tenants(ctx context.Context) ([]Tenant, error) {
	values, err := s.azure.Tenants(s.context(ctx))
	return normalizeListResult(values, err)
}

func (s *Service) Subscriptions(ctx context.Context, tenantID string) ([]Subscription, error) {
	values, err := s.azure.Subscriptions(s.context(ctx), tenantID)
	return normalizeListResult(values, err)
}

func (s *Service) ResourceGroups(ctx context.Context, tenantID, subscriptionID string) ([]ResourceGroup, error) {
	values, err := s.azure.ResourceGroups(s.context(ctx), tenantID, subscriptionID)
	return normalizeListResult(values, err)
}

func (s *Service) ModelResources(ctx context.Context, tenantID, subscriptionID, resourceGroup string) ([]ModelResource, error) {
	values, err := s.azure.ModelResources(s.context(ctx), tenantID, subscriptionID, resourceGroup)
	return normalizeListResult(values, err)
}

func (s *Service) Deployments(ctx context.Context, tenantID, subscriptionID, resourceGroup, resourceName string) ([]Deployment, error) {
	values, err := s.azure.Deployments(s.context(ctx), tenantID, subscriptionID, resourceGroup, resourceName)
	return normalizeListResult(values, err)
}

func (s *Service) Models(ctx context.Context, tenantID, subscriptionID, resourceGroup, resourceName string) ([]DeployableModel, error) {
	values, err := s.azure.Models(s.context(ctx), tenantID, subscriptionID, resourceGroup, resourceName)
	return normalizeListResult(values, err)
}

func normalizeListResult[T any](values []T, err error) ([]T, error) {
	if err != nil {
		return nil, err
	}
	if values == nil {
		return []T{}, nil
	}
	return values, nil
}

func (s *Service) PrepareOpenCodex(ctx context.Context) (OpenCodexState, error) {
	return s.opencodex.Prepare(s.context(ctx))
}

func (s *Service) OpenCodexState(ctx context.Context) (OpenCodexState, error) {
	return s.opencodex.State(s.context(ctx))
}

func (s *Service) Sync(ctx context.Context, request SyncRequest) SyncResult {
	ctx = s.context(ctx)
	result := SyncResult{ProviderID: request.ProviderID, Deployment: request.DefaultDeploymentName}
	if !s.mu.TryLock() {
		return failSync(result, "sync", "Sync is already running")
	}
	defer s.mu.Unlock()

	if err := validateSyncRequest(request); err != nil {
		return failSync(result, "validate", err.Error())
	}
	defaultDeploymentName := strings.TrimSpace(request.DefaultDeploymentName)
	result.Deployment = defaultDeploymentName
	resource, err := s.azure.GetModelResource(ctx, request.TenantID, request.SubscriptionID, request.ResourceGroup, request.ResourceName)
	if err != nil {
		return failSync(result, "azure-resource", safeErrorMessage(err))
	}
	if resource.DisableLocalAuth {
		return failSync(result, "azure-resource", "local authentication is disabled for this Azure Model Resource; opencodex API-key Sync is unavailable")
	}
	if strings.TrimSpace(resource.Endpoint) == "" {
		return failSync(result, "azure-resource", "Azure Model Resource did not return an endpoint")
	}
	deployments, err := s.azure.Deployments(ctx, request.TenantID, request.SubscriptionID, request.ResourceGroup, request.ResourceName)
	if err != nil {
		return failSync(result, "azure-deployments", safeErrorMessage(err))
	}
	models, err := s.azure.Models(ctx, request.TenantID, request.SubscriptionID, request.ResourceGroup, request.ResourceName)
	if err != nil {
		return failSync(result, "azure-models", safeErrorMessage(err))
	}
	deploymentNames := normalizeDeploymentInput(request.DeploymentNames, defaultDeploymentName)
	requested := make(map[string]struct{}, len(deploymentNames))
	for _, name := range deploymentNames {
		requested[name] = struct{}{}
	}
	orderedDeployments := make([]Deployment, 0, len(deploymentNames))
	for _, deployment := range deployments {
		if _, ok := requested[deployment.Name]; ok {
			orderedDeployments = append(orderedDeployments, deployment)
			delete(requested, deployment.Name)
		}
	}
	if len(requested) != 0 {
		return failSync(result, "azure-deployment", "one or more selected deployments do not exist in the Azure Model Resource")
	}
	for _, deployment := range orderedDeployments {
		if !deploymentIsCodexCandidate(deployment, models) {
			return failSync(result, "azure-deployment", "one or more selected deployments are not supported by the opencodex Codex route")
		}
	}
	deploymentNames = make([]string, 0, len(orderedDeployments))
	for _, deployment := range orderedDeployments {
		deploymentNames = append(deploymentNames, deployment.Name)
	}
	settings, err := s.settings.Load()
	if err != nil {
		return failSync(result, "settings", "Bridge settings could not be read")
	}
	normalizeSettings(&settings)
	providerID := strings.TrimSpace(request.ProviderID)
	if providerID == "" {
		providerID = settings.ManagedProviders[resource.ID].ProviderID
		if providerID == "" {
			providerID = defaultProviderID(resource.Name)
		}
	}
	if !validProviderID(providerID) {
		return failSync(result, "validate", "Provider ID may contain only letters, numbers, dot, underscore, and hyphen")
	}
	result.ProviderID = providerID

	state, err := s.opencodex.State(ctx)
	if err != nil && !state.NodeInstalled {
		return failSync(result, "opencodex", "Node.js 18 or newer and npm are required before preparing opencodex")
	}
	if !state.NodeInfo.Compatible {
		return failSync(result, "opencodex", "Node.js 18 or newer and npm are required before preparing opencodex")
	}
	if !state.Installed {
		return failSync(result, "opencodex", "opencodex is not installed; use Prepare opencodex first")
	}

	existingManaged := settings.ManagedProviders[resource.ID]
	if existingManaged.ProviderID != "" && existingManaged.ProviderID != providerID {
		return failSync(result, "provider", "a different Provider ID is already managed for this Azure Model Resource")
	}
	providers, err := s.opencodex.Providers(ctx)
	if err != nil {
		return failSync(result, "provider", safeErrorMessage(err))
	}
	for _, provider := range providers {
		if provider.Name == providerID && existingManaged.ProviderID == "" {
			return failSync(result, "provider", "Provider ID is already used by an unmanaged opencodex Provider")
		}
	}
	result.Stages = pendingSyncStages()

	if err := s.runStage(&result, "service", "Ensure opencodex service", func() error {
		return s.opencodex.EnsureService(ctx)
	}); err != nil {
		return result
	}
	baseURL := openAIBaseURL(resource.Endpoint)
	if err := s.runStage(&result, "provider", "Configure Bridge-managed Provider", func() error {
		return s.opencodex.EnsureProvider(ctx, providerID, baseURL, defaultDeploymentName)
	}); err != nil {
		return result
	}
	customModels := append([]string{}, deploymentNames...)
	settings.Selection = Selection{
		TenantID: request.TenantID, SubscriptionID: request.SubscriptionID, ResourceGroup: request.ResourceGroup,
		ResourceID: resource.ID, ResourceName: resource.Name, DeploymentNames: append([]string{}, deploymentNames...), DefaultDeploymentName: defaultDeploymentName, ProviderID: providerID,
	}
	settings.ManagedProviders[resource.ID] = ManagedProvider{ResourceID: resource.ID, ProviderID: providerID, TenantID: request.TenantID, SubscriptionID: request.SubscriptionID, ResourceGroup: request.ResourceGroup, ResourceName: resource.Name, Location: resource.Location, DeploymentNames: append([]string{}, deploymentNames...), DefaultDeploymentName: defaultDeploymentName}
	if err := s.runStage(&result, "settings", "Save Bridge-managed Provider mapping", func() error {
		if err := s.settings.Save(settings); err != nil {
			return errors.New("Bridge settings could not be saved")
		}
		return nil
	}); err != nil {
		return result
	}

	keys, err := s.azure.ListKeys(ctx, request.TenantID, request.SubscriptionID, request.ResourceGroup, request.ResourceName)
	if err != nil {
		return failSync(result, "key", safeErrorMessage(err))
	}
	primaryKey := keys.PrimaryKey
	keys.PrimaryKey = ""
	if err := s.runStage(&result, "key", "Register PrimaryKey through opencodex", func() error {
		return s.opencodex.AddPrimaryKey(ctx, providerID, primaryKey)
	}); err != nil {
		return result
	}
	primaryKey = ""

	if err := s.runStage(&result, "selected", "Select models in opencodex", func() error {
		return s.opencodex.EnsureSelectedModels(ctx, providerID, deploymentNames)
	}); err != nil {
		return result
	}
	// models add writes the disk config directly. Keep it after live-proxy mutations so
	// a stale proxy snapshot cannot overwrite the newly registered custom model.
	if err := s.runStage(&result, "model", "Register selected custom models", func() error {
		return s.opencodex.EnsureCustomModels(ctx, providerID, customModels)
	}); err != nil {
		return result
	}
	if err := s.runStage(&result, "catalog", "Sync Codex Catalog", func() error {
		return s.opencodex.SyncCatalog(ctx)
	}); err != nil {
		return result
	}
	state, err = s.opencodex.State(ctx)
	if err != nil || !state.Health.Ready {
		return failSync(result, "connection", "opencodex health did not report a ready service")
	}
	connectionFailed := false
	for _, deployment := range deploymentNames {
		connection := ConnectionResult{Deployment: deployment, Status: "succeeded", Message: "Responses endpoint is ready"}
		if err := s.opencodex.TestResponse(ctx, state.Health.Port, providerID, deployment); err != nil {
			connection.Status = "failed"
			connection.Message = safeErrorMessage(err)
			connectionFailed = true
		}
		result.Connections = append(result.Connections, connection)
	}
	if connectionFailed {
		setSyncStage(&result, "connection", "failed", "one or more Responses endpoint tests failed")
		return result
	}
	setSyncStage(&result, "connection", "succeeded", "Test Responses endpoint")
	result.OK = true
	return result
}

func (s *Service) runStage(result *SyncResult, name, description string, action func() error) error {
	if err := action(); err != nil {
		setSyncStage(result, name, "failed", safeErrorMessage(err))
		return err
	}
	setSyncStage(result, name, "succeeded", description)
	return nil
}

func pendingSyncStages() []SyncStage {
	return []SyncStage{
		{Name: "service", Status: "pending", Message: "Ensure opencodex service"},
		{Name: "provider", Status: "pending", Message: "Configure Bridge-managed Provider"},
		{Name: "settings", Status: "pending", Message: "Save Bridge-managed Provider mapping"},
		{Name: "key", Status: "pending", Message: "Register PrimaryKey through opencodex"},
		{Name: "selected", Status: "pending", Message: "Select models in opencodex"},
		{Name: "model", Status: "pending", Message: "Register selected custom models"},
		{Name: "catalog", Status: "pending", Message: "Sync Codex Catalog"},
		{Name: "connection", Status: "pending", Message: "Test Responses endpoints"},
	}
}

func setSyncStage(result *SyncResult, name, status, message string) {
	for index := range result.Stages {
		if result.Stages[index].Name == name {
			result.Stages[index].Status = status
			result.Stages[index].Message = message
			return
		}
	}
	result.Stages = append(result.Stages, SyncStage{Name: name, Status: status, Message: message})
}

func validateSyncRequest(request SyncRequest) error {
	for name, value := range map[string]string{
		"tenant":               request.TenantID,
		"subscription":         request.SubscriptionID,
		"resource group":       request.ResourceGroup,
		"Azure Model Resource": request.ResourceName,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must be selected", name)
		}
	}
	if !request.ConfirmCosts {
		return errors.New("Sync sends a real Responses request and may incur Azure charges; confirm the warning first")
	}
	names := normalizeDeploymentInput(request.DeploymentNames, request.DefaultDeploymentName)
	if len(names) == 0 {
		return errors.New("at least one deployment must be selected")
	}
	defaultName := strings.TrimSpace(request.DefaultDeploymentName)
	for _, name := range names {
		if name == defaultName {
			return nil
		}
	}
	return errors.New("default deployment must be included in the selected deployments")
}

func normalizeDeploymentInput(names []string, defaultName string) []string {
	result := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	if len(result) == 0 && strings.TrimSpace(defaultName) != "" {
		return []string{strings.TrimSpace(defaultName)}
	}
	return result
}

func failSync(result SyncResult, name, message string) SyncResult {
	result.OK = false
	setSyncStage(&result, name, "failed", message)
	return result
}

func safeErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if operation, ok := err.(*azure.OperationError); ok {
		if operation.Scope == "" {
			return operation.Error()
		}
		return fmt.Sprintf("%s (%s)", operation.Error(), operation.Scope)
	}
	return err.Error()
}

func defaultProviderID(resourceName string) string {
	value := strings.ToLower(strings.TrimSpace(resourceName))
	var builder strings.Builder
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') || character == '.' || character == '_' || character == '-' {
			builder.WriteRune(character)
		} else {
			builder.WriteRune('-')
		}
	}
	return "az-" + strings.Trim(builder.String(), "-.")
}

func validProviderID(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') || character == '.' || character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func openAIBaseURL(endpoint string) string {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.HasSuffix(strings.ToLower(endpoint), "/openai") {
		return endpoint
	}
	return endpoint + "/openai"
}

func deploymentIsCodexCandidate(deployment Deployment, models []DeployableModel) bool {
	for _, model := range models {
		if !model.CodexCandidate || !strings.EqualFold(model.Name, deployment.ModelName) {
			continue
		}
		if deployment.ModelFormat != "" && !strings.EqualFold(model.Format, deployment.ModelFormat) {
			continue
		}
		if deployment.ModelVersion != "" && model.Version != deployment.ModelVersion {
			continue
		}
		return true
	}
	return false
}

func NewAzureClient() azure.Client {
	return azure.NewSDKClient()
}

var _ opencodex.ManagerAPI = (*opencodex.Manager)(nil)
