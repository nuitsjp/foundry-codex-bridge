package azure

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
)

type SDKClient struct {
	runner            CLIRunner
	authState         AuthState
	credentialFactory func(string, string) (azcore.TokenCredential, error)
	clientOptions     *arm.ClientOptions
}

var _ Client = (*SDKClient)(nil)
var _ DeploymentWriter = (*SDKClient)(nil)

func NewSDKClient() *SDKClient {
	return NewSDKClientWithRunner(localCLIRunner{})
}

func NewSDKClientWithRunner(runner CLIRunner) *SDKClient {
	return &SDKClient{runner: runner}
}

func (c *SDKClient) AuthState(ctx context.Context) AuthState {
	state := c.readCLIState(ctx)
	c.authState = state
	return state
}

func (c *SDKClient) Authenticate(ctx context.Context) (AuthState, error) {
	state := c.readCLIState(ctx)
	c.authState = state
	if !state.CLIInstalled {
		return state, ErrAzureCLINotInstalled
	}
	_, err := c.runner.Run(ctx, "login", "--allow-no-subscriptions", "--output", "none", "--only-show-errors")
	if err != nil {
		return state, errors.New("Azure CLI sign-in failed")
	}
	state = c.readCLIState(ctx)
	c.authState = state
	if !state.SignedIn {
		return state, errors.New("Azure CLI sign-in did not produce an active account")
	}
	return state, nil
}

func (c *SDKClient) ready() error {
	if !c.authState.CLIInstalled {
		return ErrAzureCLINotInstalled
	}
	if !c.authState.SignedIn {
		return errors.New("Azure CLI sign-in is required")
	}
	return nil
}

func (c *SDKClient) credentialForScope(tenantID, subscriptionID string) (azcore.TokenCredential, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	if c.credentialFactory != nil {
		return c.credentialFactory(tenantID, subscriptionID)
	}
	return azidentity.NewAzureCLICredential(credentialOptionsForScope(tenantID, subscriptionID))
}

func credentialOptionsForScope(tenantID, subscriptionID string) *azidentity.AzureCLICredentialOptions {
	subscriptionID = strings.TrimSpace(subscriptionID)
	if subscriptionID != "" {
		return &azidentity.AzureCLICredentialOptions{Subscription: subscriptionID}
	}
	return &azidentity.AzureCLICredentialOptions{TenantID: strings.TrimSpace(tenantID)}
}

func (c *SDKClient) readCLIState(ctx context.Context) AuthState {
	versionResult, err := c.runner.Run(ctx, "version", "--output", "json")
	if err != nil {
		return AuthState{Message: "Azure CLI is not installed"}
	}
	state := AuthState{CLIInstalled: true, Message: "Azure CLI sign-in is required"}
	var versions struct {
		AzureCLI string `json:"azure-cli"`
	}
	if json.Unmarshal([]byte(versionResult.Stdout), &versions) == nil {
		state.CLIVersion = versions.AzureCLI
	}
	accountResult, err := c.runner.Run(ctx, "account", "show", "--output", "json", "--only-show-errors")
	if err != nil {
		return state
	}
	var account struct {
		TenantID string `json:"tenantId"`
		User     struct {
			Name string `json:"name"`
		} `json:"user"`
	}
	if json.Unmarshal([]byte(accountResult.Stdout), &account) != nil || strings.TrimSpace(account.TenantID) == "" {
		state.Message = "Azure CLI returned an invalid active account"
		return state
	}
	state.SignedIn = true
	state.Username = account.User.Name
	state.TenantID = account.TenantID
	state.Message = "Azure CLI is signed in"
	return state
}

func (c *SDKClient) Tenants(ctx context.Context) ([]Tenant, error) {
	credential, err := c.credentialForScope("", "")
	if err != nil {
		return nil, err
	}
	client, err := armsubscription.NewTenantsClient(credential, c.clientOptions)
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(nil)
	var result []Tenant
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, RequireOperation(err, "List tenants", "tenant")
		}
		for _, item := range page.Value {
			id := stringValue(item.TenantID)
			result = append(result, Tenant{ID: id, DisplayName: id})
		}
	}
	return result, nil
}

func (c *SDKClient) Subscriptions(ctx context.Context, tenantID string) ([]Subscription, error) {
	credential, err := c.credentialForScope(tenantID, "")
	if err != nil {
		return nil, err
	}
	client, err := armsubscription.NewSubscriptionsClient(credential, c.clientOptions)
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(nil)
	var result []Subscription
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, RequireOperation(err, "List subscriptions", "subscription")
		}
		for _, item := range page.Value {
			result = append(result, Subscription{
				ID:       stringValue(item.SubscriptionID),
				Name:     stringValue(item.DisplayName),
				TenantID: tenantID,
				State:    subscriptionState(item.State),
			})
		}
	}
	return result, nil
}

func (c *SDKClient) ResourceGroups(ctx context.Context, tenantID, subscriptionID string) ([]ResourceGroup, error) {
	credential, err := c.credentialForScope(tenantID, subscriptionID)
	if err != nil {
		return nil, err
	}
	client, err := armcognitiveservices.NewAccountsClient(subscriptionID, credential, c.clientOptions)
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(nil)
	var accounts []*armcognitiveservices.Account
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, RequireOperation(err, "List Azure Model Resources", subscriptionID)
		}
		accounts = append(accounts, page.Value...)
	}
	return modelResourceGroups(accounts), nil
}

func (c *SDKClient) ModelResources(ctx context.Context, tenantID, subscriptionID, resourceGroup string) ([]ModelResource, error) {
	credential, err := c.credentialForScope(tenantID, subscriptionID)
	if err != nil {
		return nil, err
	}
	client, err := armcognitiveservices.NewAccountsClient(subscriptionID, credential, c.clientOptions)
	if err != nil {
		return nil, err
	}
	pager := client.NewListByResourceGroupPager(resourceGroup, nil)
	var result []ModelResource
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, RequireOperation(err, "List Azure Model Resources", subscriptionID+"/"+resourceGroup)
		}
		for _, item := range page.Value {
			if item == nil || !isSupportedKind(stringValue(item.Kind)) {
				continue
			}
			result = append(result, accountToModelResource(item))
		}
	}
	return result, nil
}

func (c *SDKClient) GetModelResource(ctx context.Context, tenantID, subscriptionID, resourceGroup, resourceName string) (ModelResource, error) {
	credential, err := c.credentialForScope(tenantID, subscriptionID)
	if err != nil {
		return ModelResource{}, err
	}
	client, err := armcognitiveservices.NewAccountsClient(subscriptionID, credential, c.clientOptions)
	if err != nil {
		return ModelResource{}, err
	}
	response, err := client.Get(ctx, resourceGroup, resourceName, nil)
	if err != nil {
		return ModelResource{}, RequireOperation(err, "Get Azure Model Resource", resourceGroup+"/"+resourceName)
	}
	if !isSupportedKind(stringValue(response.Kind)) {
		return ModelResource{}, errors.New("selected resource is not an Azure Model Resource")
	}
	return accountToModelResource(&response.Account), nil
}

func (c *SDKClient) Deployments(ctx context.Context, tenantID, subscriptionID, resourceGroup, resourceName string) ([]Deployment, error) {
	credential, err := c.credentialForScope(tenantID, subscriptionID)
	if err != nil {
		return nil, err
	}
	client, err := armcognitiveservices.NewDeploymentsClient(subscriptionID, credential, c.clientOptions)
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(resourceGroup, resourceName, nil)
	var result []Deployment
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, RequireOperation(err, "List Model Deployments", resourceGroup+"/"+resourceName)
		}
		for _, item := range page.Value {
			if item != nil {
				result = append(result, deploymentToDTO(item))
			}
		}
	}
	return result, nil
}

func (c *SDKClient) Models(ctx context.Context, tenantID, subscriptionID, resourceGroup, resourceName string) ([]DeployableModel, error) {
	credential, err := c.credentialForScope(tenantID, subscriptionID)
	if err != nil {
		return nil, err
	}
	client, err := armcognitiveservices.NewAccountsClient(subscriptionID, credential, c.clientOptions)
	if err != nil {
		return nil, err
	}
	pager := client.NewListModelsPager(resourceGroup, resourceName, nil)
	var result []DeployableModel
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, RequireOperation(err, "List deployable models", resourceGroup+"/"+resourceName)
		}
		for _, item := range page.Value {
			if item == nil {
				continue
			}
			result = append(result, accountModelToDTO(item))
		}
	}
	return result, nil
}

func (c *SDKClient) ListKeys(ctx context.Context, tenantID, subscriptionID, resourceGroup, resourceName string) (KeyBundle, error) {
	credential, err := c.credentialForScope(tenantID, subscriptionID)
	if err != nil {
		return KeyBundle{}, err
	}
	client, err := armcognitiveservices.NewAccountsClient(subscriptionID, credential, c.clientOptions)
	if err != nil {
		return KeyBundle{}, err
	}
	response, err := client.ListKeys(ctx, resourceGroup, resourceName, nil)
	if err != nil {
		return KeyBundle{}, RequireOperation(err, "List Azure Model Resource keys", resourceGroup+"/"+resourceName)
	}
	return KeyBundle{PrimaryKey: stringValue(response.Key1)}, nil
}

func (c *SDKClient) CreateOrUpdateDeployment(ctx context.Context, tenantID, subscriptionID, resourceGroup, resourceName string, input DeploymentInput) (Deployment, error) {
	credential, err := c.credentialForScope(tenantID, subscriptionID)
	if err != nil {
		return Deployment{}, err
	}
	client, err := armcognitiveservices.NewDeploymentsClient(subscriptionID, credential, deploymentClientOptions(c.clientOptions))
	if err != nil {
		return Deployment{}, err
	}

	deployment := deploymentFromInput(input)
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroup, resourceName, input.Name, deployment, nil)
	if err != nil {
		return Deployment{}, RequireOperation(err, "Create or update Model Deployment", deploymentScope(resourceGroup, resourceName, input.Name))
	}
	response, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return Deployment{}, RequireOperation(err, "Create or update Model Deployment", deploymentScope(resourceGroup, resourceName, input.Name))
	}
	return deploymentToDTO(&response.Deployment), nil
}

func isSupportedKind(kind string) bool {
	return strings.EqualFold(kind, "AIServices") || strings.EqualFold(kind, "OpenAI")
}

func modelResourceGroups(accounts []*armcognitiveservices.Account) []ResourceGroup {
	groupsByName := make(map[string]ResourceGroup)
	for _, account := range accounts {
		if account == nil || !isSupportedKind(stringValue(account.Kind)) {
			continue
		}
		resourceID, err := arm.ParseResourceID(stringValue(account.ID))
		if err != nil || resourceID.ResourceGroupName == "" {
			continue
		}
		key := strings.ToLower(resourceID.ResourceGroupName)
		if _, exists := groupsByName[key]; !exists {
			groupsByName[key] = ResourceGroup{Name: resourceID.ResourceGroupName}
		}
	}

	result := make([]ResourceGroup, 0, len(groupsByName))
	for _, group := range groupsByName {
		result = append(result, group)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name)
	})
	return result
}

func isCodexCandidate(format string, capabilities map[string]string) bool {
	key := "agentsV2"
	if strings.EqualFold(format, "OpenAI") {
		key = "responses"
	}
	return strings.EqualFold(capabilities[key], "true")
}

func accountToModelResource(item *armcognitiveservices.Account) ModelResource {
	resource := ModelResource{
		ID:       stringValue(item.ID),
		Name:     stringValue(item.Name),
		Kind:     stringValue(item.Kind),
		Location: stringValue(item.Location),
	}
	if item.Properties != nil {
		resource.Endpoint = stringValue(item.Properties.Endpoint)
		resource.DisableLocalAuth = boolValue(item.Properties.DisableLocalAuth)
	}
	return resource
}

func deploymentToDTO(item *armcognitiveservices.Deployment) Deployment {
	result := Deployment{ID: stringValue(item.ID), Name: stringValue(item.Name)}
	if item.Properties != nil {
		result.ProvisioningState = deploymentState(item.Properties.ProvisioningState)
		if item.Properties.VersionUpgradeOption != nil {
			result.VersionUpgradeOption = string(*item.Properties.VersionUpgradeOption)
		}
		if item.Properties.Model != nil {
			result.ModelName = stringValue(item.Properties.Model.Name)
			result.ModelFormat = stringValue(item.Properties.Model.Format)
			result.ModelVersion = stringValue(item.Properties.Model.Version)
		}
	}
	if item.SKU != nil {
		result.SKU = stringValue(item.SKU.Name)
		if item.SKU.Capacity != nil {
			result.Capacity = *item.SKU.Capacity
		}
	}
	return result
}

func accountModelToDTO(item *armcognitiveservices.AccountModel) DeployableModel {
	capabilities := make(map[string]string, len(item.Capabilities))
	for key, value := range item.Capabilities {
		capabilities[key] = stringValue(value)
	}
	format := stringValue(item.Format)
	result := DeployableModel{
		Name:           stringValue(item.Name),
		Format:         format,
		Version:        stringValue(item.Version),
		Capabilities:   capabilities,
		MaxCapacity:    int32Value(item.MaxCapacity),
		CodexCandidate: isCodexCandidate(format, capabilities),
		SKUs:           make([]ModelSKU, 0, len(item.SKUs)),
	}
	for _, sku := range item.SKUs {
		if sku == nil {
			continue
		}
		result.SKUs = append(result.SKUs, modelSKUToDTO(sku))
	}
	return result
}

func modelSKUToDTO(item *armcognitiveservices.ModelSKU) ModelSKU {
	result := ModelSKU{
		Name:      stringValue(item.Name),
		UsageName: stringValue(item.UsageName),
		Unit:      capacityUnit(stringValue(item.Name), stringValue(item.UsageName)),
	}
	if item.Capacity != nil {
		result.Capacity = CapacityConstraints{
			Minimum:       int32Value(item.Capacity.Minimum),
			Maximum:       int32Value(item.Capacity.Maximum),
			Step:          int32Value(item.Capacity.Step),
			Default:       int32Value(item.Capacity.Default),
			AllowedValues: int32Values(item.Capacity.AllowedValues),
		}
	}
	return result
}

func deploymentFromInput(input DeploymentInput) armcognitiveservices.Deployment {
	properties := &armcognitiveservices.DeploymentProperties{
		Model: &armcognitiveservices.DeploymentModel{
			Name:    stringPointer(input.ModelName),
			Format:  stringPointer(input.ModelFormat),
			Version: stringPointer(input.ModelVersion),
		},
	}
	if input.VersionUpgradeOption != nil {
		option := armcognitiveservices.DeploymentModelVersionUpgradeOption(*input.VersionUpgradeOption)
		properties.VersionUpgradeOption = &option
	}
	return armcognitiveservices.Deployment{
		Properties: properties,
		SKU: &armcognitiveservices.SKU{
			Name:     stringPointer(input.SKU),
			Capacity: &input.Capacity,
		},
	}
}

func deploymentScope(resourceGroup, resourceName, deploymentName string) string {
	return strings.Trim(strings.Join([]string{resourceGroup, resourceName, deploymentName}, "/"), "/")
}

func deploymentClientOptions(options *arm.ClientOptions) *arm.ClientOptions {
	if options == nil {
		options = &arm.ClientOptions{}
	} else {
		options = options.Clone()
	}
	options.Retry.MaxRetries = -1
	return options
}

func capacityUnit(name, usageName string) string {
	value := strings.ToLower(name + " " + usageName)
	if strings.Contains(value, "provisioned") || strings.Contains(value, "ptu") {
		return "PTU"
	}
	return "TPM"
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPointer(value string) *string {
	return &value
}

func int32Value(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

func int32Values(values []*int32) []int32 {
	result := make([]int32, 0, len(values))
	for _, value := range values {
		if value != nil {
			result = append(result, *value)
		}
	}
	return result
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func subscriptionState(value *armsubscription.SubscriptionState) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func deploymentState(value *armcognitiveservices.DeploymentProvisioningState) string {
	if value == nil {
		return ""
	}
	return string(*value)
}
