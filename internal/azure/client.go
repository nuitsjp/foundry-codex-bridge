package azure

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity/cache"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
)

var ErrClientNotConfigured = errors.New("Azure client ID is not configured")

type SDKClient struct {
	dataDir        string
	clientID       string
	credential     *azidentity.InteractiveBrowserCredential
	cache          azidentity.Cache
	authRecord     azidentity.AuthenticationRecord
	authState      AuthState
	initialization error
}

func NewSDKClient(dataDir, clientID string) *SDKClient {
	client := &SDKClient{dataDir: dataDir, clientID: strings.TrimSpace(clientID)}
	if client.clientID == "" {
		client.initialization = ErrClientNotConfigured
		return client
	}

	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		client.initialization = err
		return client
	}
	persistentCache, err := cache.New(nil)
	if err != nil {
		client.initialization = err
		return client
	}
	client.cache = persistentCache
	client.authRecord = client.loadAuthRecord()
	credential, err := azidentity.NewInteractiveBrowserCredential(&azidentity.InteractiveBrowserCredentialOptions{
		ClientID:             client.clientID,
		TenantID:             "organizations",
		AuthenticationRecord: client.authRecord,
		Cache:                persistentCache,
	})
	if err != nil {
		client.initialization = err
		return client
	}
	client.credential = credential
	if client.authRecord.Username != "" {
		client.authState = AuthState{
			SignedIn: true,
			Username: client.authRecord.Username,
			TenantID: client.authRecord.TenantID,
		}
	}
	return client
}

func (c *SDKClient) AuthState() AuthState {
	return c.authState
}

func (c *SDKClient) Authenticate(ctx context.Context) (AuthState, error) {
	if c.initialization != nil {
		return AuthState{}, c.initialization
	}
	record, err := c.credential.Authenticate(ctx, nil)
	if err != nil {
		return AuthState{}, err
	}
	if err := c.saveAuthRecord(record); err != nil {
		return AuthState{}, err
	}
	c.authRecord = record
	c.authState = AuthState{SignedIn: true, Username: record.Username, TenantID: record.TenantID}
	return c.authState, nil
}

func (c *SDKClient) SignOut(_ context.Context) error {
	c.authRecord = azidentity.AuthenticationRecord{}
	c.authState = AuthState{}
	if c.dataDir == "" {
		return nil
	}
	if err := os.Remove(c.authRecordPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (c *SDKClient) ready() error {
	if c.initialization != nil {
		return c.initialization
	}
	if c.credential == nil || !c.authState.SignedIn {
		return errors.New("Azure sign-in is required")
	}
	return nil
}

func (c *SDKClient) loadAuthRecord() azidentity.AuthenticationRecord {
	data, err := os.ReadFile(c.authRecordPath())
	if err != nil {
		return azidentity.AuthenticationRecord{}
	}
	var record azidentity.AuthenticationRecord
	if json.Unmarshal(data, &record) != nil {
		return azidentity.AuthenticationRecord{}
	}
	return record
}

func (c *SDKClient) saveAuthRecord(record azidentity.AuthenticationRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(c.dataDir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(c.authRecordPath(), data, 0o600)
}

func (c *SDKClient) authRecordPath() string {
	return filepath.Join(c.dataDir, "authentication-record.json")
}

func (c *SDKClient) Tenants(ctx context.Context) ([]Tenant, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	client, err := armsubscription.NewTenantsClient(c.credential, nil)
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
	if err := c.ready(); err != nil {
		return nil, err
	}
	client, err := armsubscription.NewSubscriptionsClient(c.credential, nil)
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

func (c *SDKClient) ResourceGroups(ctx context.Context, subscriptionID string) ([]ResourceGroup, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	client, err := armresources.NewResourceGroupsClient(subscriptionID, c.credential, nil)
	if err != nil {
		return nil, err
	}
	pager := client.NewListPager(nil)
	var result []ResourceGroup
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, RequireOperation(err, "List resource groups", subscriptionID)
		}
		for _, item := range page.Value {
			result = append(result, ResourceGroup{Name: stringValue(item.Name)})
		}
	}
	return result, nil
}

func (c *SDKClient) ModelResources(ctx context.Context, subscriptionID, resourceGroup string) ([]ModelResource, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	client, err := armcognitiveservices.NewAccountsClient(subscriptionID, c.credential, nil)
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

func (c *SDKClient) GetModelResource(ctx context.Context, subscriptionID, resourceGroup, resourceName string) (ModelResource, error) {
	if err := c.ready(); err != nil {
		return ModelResource{}, err
	}
	client, err := armcognitiveservices.NewAccountsClient(subscriptionID, c.credential, nil)
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

func (c *SDKClient) Deployments(ctx context.Context, subscriptionID, resourceGroup, resourceName string) ([]Deployment, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	client, err := armcognitiveservices.NewDeploymentsClient(subscriptionID, c.credential, nil)
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

func (c *SDKClient) Models(ctx context.Context, subscriptionID, resourceGroup, resourceName string) ([]DeployableModel, error) {
	if err := c.ready(); err != nil {
		return nil, err
	}
	client, err := armcognitiveservices.NewAccountsClient(subscriptionID, c.credential, nil)
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
			capabilities := make(map[string]string, len(item.Capabilities))
			for key, value := range item.Capabilities {
				capabilities[key] = stringValue(value)
			}
			format := stringValue(item.Format)
			result = append(result, DeployableModel{
				Name:           stringValue(item.Name),
				Format:         format,
				Version:        stringValue(item.Version),
				Capabilities:   capabilities,
				CodexCandidate: isCodexCandidate(format, capabilities),
			})
		}
	}
	return result, nil
}

func (c *SDKClient) ListKeys(ctx context.Context, subscriptionID, resourceGroup, resourceName string) (KeyBundle, error) {
	if err := c.ready(); err != nil {
		return KeyBundle{}, err
	}
	client, err := armcognitiveservices.NewAccountsClient(subscriptionID, c.credential, nil)
	if err != nil {
		return KeyBundle{}, err
	}
	response, err := client.ListKeys(ctx, resourceGroup, resourceName, nil)
	if err != nil {
		return KeyBundle{}, RequireOperation(err, "List Azure Model Resource keys", resourceGroup+"/"+resourceName)
	}
	return KeyBundle{PrimaryKey: stringValue(response.Key1)}, nil
}

func isSupportedKind(kind string) bool {
	return strings.EqualFold(kind, "AIServices") || strings.EqualFold(kind, "OpenAI")
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

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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
