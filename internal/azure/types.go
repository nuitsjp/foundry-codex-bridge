package azure

import "context"

type AuthState struct {
	CLIInstalled bool   `json:"cliInstalled"`
	CLIVersion   string `json:"cliVersion"`
	SignedIn     bool   `json:"signedIn"`
	Username     string `json:"username"`
	TenantID     string `json:"tenantId"`
	Message      string `json:"message"`
}

type Tenant struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type Subscription struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	TenantID string `json:"tenantId"`
	State    string `json:"state"`
}

type ResourceGroup struct {
	Name     string `json:"name"`
	Location string `json:"location"`
}

type ModelResource struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	Location         string `json:"location"`
	Endpoint         string `json:"endpoint"`
	DisableLocalAuth bool   `json:"disableLocalAuth"`
}

type Deployment struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	ModelName            string `json:"modelName"`
	ModelFormat          string `json:"modelFormat"`
	ModelVersion         string `json:"modelVersion"`
	SKU                  string `json:"sku"`
	Capacity             int32  `json:"capacity"`
	VersionUpgradeOption string `json:"versionUpgradeOption"`
	ProvisioningState    string `json:"provisioningState"`
}

type CapacityConstraints struct {
	Minimum       int32   `json:"minimum"`
	Maximum       int32   `json:"maximum"`
	Step          int32   `json:"step"`
	Default       int32   `json:"default"`
	AllowedValues []int32 `json:"allowedValues"`
}

type ModelSKU struct {
	Name      string              `json:"name"`
	UsageName string              `json:"usageName"`
	Capacity  CapacityConstraints `json:"capacity"`
	Unit      string              `json:"unit"`
}

type DeployableModel struct {
	Name           string            `json:"name"`
	Format         string            `json:"format"`
	Version        string            `json:"version"`
	Capabilities   map[string]string `json:"capabilities"`
	SKUs           []ModelSKU        `json:"skus"`
	MaxCapacity    int32             `json:"maxCapacity"`
	CodexCandidate bool              `json:"codexCandidate"`
}

type DeploymentInput struct {
	Name                 string  `json:"name"`
	ModelName            string  `json:"modelName"`
	ModelFormat          string  `json:"modelFormat"`
	ModelVersion         string  `json:"modelVersion"`
	SKU                  string  `json:"sku"`
	Capacity             int32   `json:"capacity"`
	VersionUpgradeOption *string `json:"versionUpgradeOption"`
}

type DeploymentWriter interface {
	CreateOrUpdateDeployment(context.Context, string, string, string, string, DeploymentInput) (Deployment, error)
}

// KeyBundle never leaves the bridge service. In particular, it is not a Wails
// DTO and is not persisted in settings.
type KeyBundle struct {
	PrimaryKey string
}

type Client interface {
	Authenticate(context.Context) (AuthState, error)
	AuthState(context.Context) AuthState
	Tenants(context.Context) ([]Tenant, error)
	Subscriptions(context.Context, string) ([]Subscription, error)
	ResourceGroups(context.Context, string, string) ([]ResourceGroup, error)
	ModelResources(context.Context, string, string, string) ([]ModelResource, error)
	Deployments(context.Context, string, string, string, string) ([]Deployment, error)
	Models(context.Context, string, string, string, string) ([]DeployableModel, error)
	GetModelResource(context.Context, string, string, string, string) (ModelResource, error)
	ListKeys(context.Context, string, string, string, string) (KeyBundle, error)
}
