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
		settings = Settings{ManagedProviders: map[string]string{}}
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
	return s.azure.Tenants(s.context(ctx))
}

func (s *Service) Subscriptions(ctx context.Context, tenantID string) ([]Subscription, error) {
	return s.azure.Subscriptions(s.context(ctx), tenantID)
}

func (s *Service) ResourceGroups(ctx context.Context, tenantID, subscriptionID string) ([]ResourceGroup, error) {
	return s.azure.ResourceGroups(s.context(ctx), tenantID, subscriptionID)
}

func (s *Service) ModelResources(ctx context.Context, tenantID, subscriptionID, resourceGroup string) ([]ModelResource, error) {
	return s.azure.ModelResources(s.context(ctx), tenantID, subscriptionID, resourceGroup)
}

func (s *Service) Deployments(ctx context.Context, tenantID, subscriptionID, resourceGroup, resourceName string) ([]Deployment, error) {
	return s.azure.Deployments(s.context(ctx), tenantID, subscriptionID, resourceGroup, resourceName)
}

func (s *Service) Models(ctx context.Context, tenantID, subscriptionID, resourceGroup, resourceName string) ([]DeployableModel, error) {
	return s.azure.Models(s.context(ctx), tenantID, subscriptionID, resourceGroup, resourceName)
}

func (s *Service) PrepareOpenCodex(ctx context.Context) (OpenCodexState, error) {
	return s.opencodex.Prepare(s.context(ctx))
}

func (s *Service) OpenCodexState(ctx context.Context) (OpenCodexState, error) {
	return s.opencodex.State(s.context(ctx))
}

func (s *Service) Sync(ctx context.Context, request SyncRequest) SyncResult {
	ctx = s.context(ctx)
	result := SyncResult{ProviderID: request.ProviderID, Deployment: request.DeploymentName}
	if !s.mu.TryLock() {
		return failSync(result, "sync", "Sync is already running")
	}
	defer s.mu.Unlock()

	if err := validateSyncRequest(request); err != nil {
		return failSync(result, "validate", err.Error())
	}
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
	var selectedDeployment Deployment
	for _, deployment := range deployments {
		if deployment.Name == request.DeploymentName {
			selectedDeployment = deployment
			break
		}
	}
	if selectedDeployment.Name == "" {
		return failSync(result, "azure-deployment", "the selected deployment does not exist in the Azure Model Resource")
	}
	models, err := s.azure.Models(ctx, request.TenantID, request.SubscriptionID, request.ResourceGroup, request.ResourceName)
	if err != nil {
		return failSync(result, "azure-models", safeErrorMessage(err))
	}
	if !deploymentIsCodexCandidate(selectedDeployment, models) {
		return failSync(result, "azure-deployment", "the selected deployment is not supported by the opencodex Codex route")
	}
	settings, err := s.settings.Load()
	if err != nil {
		return failSync(result, "settings", "Bridge settings could not be read")
	}
	if settings.ManagedProviders == nil {
		settings.ManagedProviders = map[string]string{}
	}
	providerID := request.ProviderID
	if providerID == "" {
		providerID = settings.ManagedProviders[resource.ID]
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

	if existing := settings.ManagedProviders[resource.ID]; existing != "" && existing != providerID {
		return failSync(result, "provider", "a different Provider ID is already managed for this Azure Model Resource")
	}
	if existing, err := s.opencodex.ProviderExists(ctx, providerID); err != nil {
		return failSync(result, "provider", safeErrorMessage(err))
	} else if existing && settings.ManagedProviders[resource.ID] == "" {
		return failSync(result, "provider", "Provider ID is already used by an unmanaged opencodex Provider")
	}

	if err := s.runStage(&result, "service", "Ensure opencodex service", func() error {
		return s.opencodex.EnsureService(ctx)
	}); err != nil {
		return result
	}
	baseURL := openAIBaseURL(resource.Endpoint)
	if err := s.runStage(&result, "provider", "Configure Bridge-managed Provider", func() error {
		return s.opencodex.EnsureProvider(ctx, providerID, baseURL, request.DeploymentName)
	}); err != nil {
		return result
	}
	settings.Selection = Selection{
		TenantID: request.TenantID, SubscriptionID: request.SubscriptionID, ResourceGroup: request.ResourceGroup,
		ResourceID: resource.ID, ResourceName: resource.Name, DeploymentName: request.DeploymentName, ProviderID: providerID,
	}
	settings.ManagedProviders[resource.ID] = providerID
	if err := s.settings.Save(settings); err != nil {
		return failSync(result, "settings", "Bridge settings could not be saved")
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

	if err := s.runStage(&result, "model", "Register selected custom model", func() error {
		return s.opencodex.EnsureCustomModel(ctx, providerID, request.DeploymentName)
	}); err != nil {
		return result
	}
	if err := s.runStage(&result, "selected", "Select model in opencodex", func() error {
		return s.opencodex.SelectModel(ctx, providerID, request.DeploymentName)
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
	if err := s.runStage(&result, "connection", "Test Responses endpoint", func() error {
		return s.opencodex.TestResponse(ctx, state.Health.Port, providerID, request.DeploymentName)
	}); err != nil {
		return result
	}
	result.OK = true
	return result
}

func (s *Service) runStage(result *SyncResult, name, description string, action func() error) error {
	if err := action(); err != nil {
		result.Stages = append(result.Stages, SyncStage{Name: name, Status: "failed", Message: safeErrorMessage(err)})
		return err
	}
	result.Stages = append(result.Stages, SyncStage{Name: name, Status: "succeeded", Message: description})
	return nil
}

func validateSyncRequest(request SyncRequest) error {
	for name, value := range map[string]string{
		"tenant":               request.TenantID,
		"subscription":         request.SubscriptionID,
		"resource group":       request.ResourceGroup,
		"Azure Model Resource": request.ResourceName,
		"deployment":           request.DeploymentName,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must be selected", name)
		}
	}
	if !request.ConfirmCosts {
		return errors.New("Sync sends a real Responses request and may incur Azure charges; confirm the warning first")
	}
	return nil
}

func failSync(result SyncResult, name, message string) SyncResult {
	result.OK = false
	result.Stages = append(result.Stages, SyncStage{Name: name, Status: "failed", Message: message})
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
