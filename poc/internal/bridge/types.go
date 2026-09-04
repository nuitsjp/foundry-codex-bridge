package bridge

import (
	"context"

	"github.com/nuitsjp/foundry-codex-bridge/internal/azure"
	"github.com/nuitsjp/foundry-codex-bridge/internal/opencodex"
)

type AuthState = azure.AuthState
type Tenant = azure.Tenant
type Subscription = azure.Subscription
type ResourceGroup = azure.ResourceGroup
type ModelResource = azure.ModelResource
type Deployment = azure.Deployment
type DeployableModel = azure.DeployableModel
type OperationError = azure.OperationError
type OpenCodexState = opencodex.State

type DeploymentRequest struct {
	TenantID             string `json:"tenantId"`
	SubscriptionID       string `json:"subscriptionId"`
	ResourceGroup        string `json:"resourceGroup"`
	ResourceName         string `json:"resourceName"`
	DeploymentName       string `json:"deploymentName"`
	ModelName            string `json:"modelName"`
	ModelFormat          string `json:"modelFormat"`
	ModelVersion         string `json:"modelVersion"`
	SKU                  string `json:"sku"`
	Capacity             int32  `json:"capacity"`
	VersionUpgradeOption string `json:"versionUpgradeOption"`
	Confirm              bool   `json:"confirm"`
}

type DeploymentOperationResult struct {
	OK           bool            `json:"ok"`
	Operation    string          `json:"operation"`
	Status       string          `json:"status"`
	Deployment   *Deployment     `json:"deployment,omitempty"`
	Error        *OperationError `json:"error,omitempty"`
	Message      string          `json:"message"`
	SyncRequired bool            `json:"syncRequired"`
}

type Selection struct {
	TenantID              string   `json:"tenantId"`
	SubscriptionID        string   `json:"subscriptionId"`
	ResourceGroup         string   `json:"resourceGroup"`
	ResourceID            string   `json:"resourceId"`
	ResourceName          string   `json:"resourceName"`
	DeploymentNames       []string `json:"deploymentNames"`
	DefaultDeploymentName string   `json:"defaultDeploymentName"`
	DeploymentName        string   `json:"-"`
	ProviderID            string   `json:"providerId"`
}

type ManagedProvider struct {
	ResourceID            string   `json:"resourceId"`
	ProviderID            string   `json:"providerId"`
	TenantID              string   `json:"tenantId"`
	SubscriptionID        string   `json:"subscriptionId"`
	ResourceGroup         string   `json:"resourceGroup"`
	ResourceName          string   `json:"resourceName"`
	Location              string   `json:"location"`
	DeploymentNames       []string `json:"deploymentNames"`
	DefaultDeploymentName string   `json:"defaultDeploymentName"`
}

type Snapshot struct {
	Auth             AuthState      `json:"auth"`
	Selection        Selection      `json:"selection"`
	OpenCodex        OpenCodexState `json:"openCodex"`
	AzureClientReady bool           `json:"azureClientReady"`
}

type SyncRequest struct {
	TenantID              string   `json:"tenantId"`
	SubscriptionID        string   `json:"subscriptionId"`
	ResourceGroup         string   `json:"resourceGroup"`
	ResourceName          string   `json:"resourceName"`
	DeploymentNames       []string `json:"deploymentNames"`
	DefaultDeploymentName string   `json:"defaultDeploymentName"`
	ProviderID            string   `json:"providerId"`
	ConfirmCosts          bool     `json:"confirmCosts"`
}

type SyncStage struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type SyncResult struct {
	OK          bool               `json:"ok"`
	ProviderID  string             `json:"providerId"`
	Deployment  string             `json:"deployment"`
	Stages      []SyncStage        `json:"stages"`
	Connections []ConnectionResult `json:"connections"`
}

type ConnectionResult struct {
	Deployment string `json:"deployment"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

type SyncChange struct {
	Area    string `json:"area"`
	Action  string `json:"action"`
	Details string `json:"details"`
}

type SyncPreview struct {
	OK         bool         `json:"ok"`
	ProviderID string       `json:"providerId"`
	Changes    []SyncChange `json:"changes"`
	Message    string       `json:"message"`
}

type DisconnectPreview struct {
	OK                   bool            `json:"ok"`
	Managed              ManagedProvider `json:"managed"`
	IsDefault            bool            `json:"isDefault"`
	DependentCombos      []string        `json:"dependentCombos"`
	ReplacementProviders []string        `json:"replacementProviders"`
	Message              string          `json:"message"`
}

type DisconnectRequest struct {
	ResourceID            string `json:"resourceId"`
	ReplacementProviderID string `json:"replacementProviderId"`
}

type ActionResult struct {
	OK     bool        `json:"ok"`
	Stages []SyncStage `json:"stages"`
}

type Settings struct {
	Selection        Selection                  `json:"selection"`
	ManagedProviders map[string]ManagedProvider `json:"managedProviders,omitempty"`
}

type SettingsStore interface {
	Load() (Settings, error)
	Save(Settings) error
}

type Service struct {
	azure     azure.Client
	opencodex opencodex.ManagerAPI
	settings  SettingsStore
	mu        syncLocker
}

type syncLocker struct{ channel chan struct{} }

func newSyncLocker() syncLocker {
	return syncLocker{channel: make(chan struct{}, 1)}
}

func (l syncLocker) TryLock() bool {
	select {
	case l.channel <- struct{}{}:
		return true
	default:
		return false
	}
}

func (l syncLocker) Unlock() { <-l.channel }

func NewService(client azure.Client, manager opencodex.ManagerAPI, store SettingsStore) *Service {
	return &Service{azure: client, opencodex: manager, settings: store, mu: newSyncLocker()}
}

func (s *Service) context(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
