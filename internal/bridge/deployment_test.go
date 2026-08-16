package bridge

import (
	"context"
	"reflect"
	"testing"

	"github.com/nuitsjp/foundry-codex-bridge/internal/azure"
)

type recordingDeploymentAzure struct {
	*fakeAzure
	inputs []azure.DeploymentInput
	err    error
}

func (f *recordingDeploymentAzure) CreateOrUpdateDeployment(_ context.Context, _, _, _, _ string, input azure.DeploymentInput) (azure.Deployment, error) {
	f.inputs = append(f.inputs, input)
	if f.err != nil {
		return azure.Deployment{}, f.err
	}
	policy := ""
	if input.VersionUpgradeOption != nil {
		policy = *input.VersionUpgradeOption
	}
	return azure.Deployment{
		Name:                 input.Name,
		ModelName:            input.ModelName,
		ModelVersion:         input.ModelVersion,
		SKU:                  input.SKU,
		Capacity:             input.Capacity,
		VersionUpgradeOption: policy,
		ProvisioningState:    "Succeeded",
	}, nil
}

func TestCreateDeploymentRequiresExplicitConfirmation(t *testing.T) {
	azureClient := &recordingDeploymentAzure{fakeAzure: newFakeAzure()}
	result := NewService(azureClient, &fakeOpenCodex{}, &fakeSettings{}).CreateDeployment(context.Background(), DeploymentRequest{
		TenantID: "tenant", SubscriptionID: "subscription", ResourceGroup: "rg", ResourceName: "account",
		DeploymentName: "new-deployment", ModelName: "gpt-4o", ModelFormat: "OpenAI", ModelVersion: "2024-11-20", SKU: "Standard", Capacity: 10,
	})
	if result.OK || result.Status != "failed" || result.Error == nil || result.Error.Class != azure.ErrorValidation {
		t.Fatalf("CreateDeployment() = %#v", result)
	}
	if len(azureClient.inputs) != 0 {
		t.Fatalf("writer called without confirmation: %#v", azureClient.inputs)
	}
}

func TestCreateDeploymentMapsInputAndMarksSyncRequiredWithoutSyncing(t *testing.T) {
	azureClient := &recordingDeploymentAzure{fakeAzure: newFakeAzure()}
	ocx := &fakeOpenCodex{}
	result := NewService(azureClient, ocx, &fakeSettings{}).CreateDeployment(context.Background(), DeploymentRequest{
		TenantID: "tenant", SubscriptionID: "subscription", ResourceGroup: "rg", ResourceName: "account",
		DeploymentName: "new-deployment", ModelName: "gpt-4o", ModelFormat: "OpenAI", ModelVersion: "2024-11-20", SKU: "Standard", Capacity: 10,
		VersionUpgradeOption: "OnceNewDefaultVersionAvailable", Confirm: true,
	})
	if !result.OK || result.Status != "succeeded" || !result.SyncRequired || result.Deployment == nil {
		t.Fatalf("CreateDeployment() = %#v", result)
	}
	wantPolicy := "OnceNewDefaultVersionAvailable"
	want := azure.DeploymentInput{Name: "new-deployment", ModelName: "gpt-4o", ModelFormat: "OpenAI", ModelVersion: "2024-11-20", SKU: "Standard", Capacity: 10, VersionUpgradeOption: &wantPolicy}
	if len(azureClient.inputs) != 1 || !reflect.DeepEqual(azureClient.inputs[0], want) {
		t.Fatalf("writer input = %#v, want %#v", azureClient.inputs, want)
	}
	if len(ocx.called) != 0 {
		t.Fatal("deployment mutation must not trigger Sync")
	}
}

func TestUpdateDeploymentPreservesUnsetValuesAndCurrentPolicy(t *testing.T) {
	currentPolicy := "NoAutoUpgrade"
	azureClient := &recordingDeploymentAzure{fakeAzure: newFakeAzure()}
	azureClient.deployments = []azure.Deployment{{Name: "existing", ModelName: "gpt-4o", ModelFormat: "OpenAI", ModelVersion: "2024-11-20", SKU: "Standard", Capacity: 10, VersionUpgradeOption: currentPolicy}}
	result := NewService(azureClient, &fakeOpenCodex{}, &fakeSettings{}).UpdateDeployment(context.Background(), DeploymentRequest{
		TenantID: "tenant", SubscriptionID: "subscription", ResourceGroup: "rg", ResourceName: "account",
		DeploymentName: "existing", Confirm: true,
	})
	if !result.OK {
		t.Fatalf("UpdateDeployment() = %#v", result)
	}
	if len(azureClient.inputs) != 1 {
		t.Fatalf("writer calls = %#v", azureClient.inputs)
	}
	input := azureClient.inputs[0]
	if input.ModelName != "gpt-4o" || input.ModelFormat != "OpenAI" || input.ModelVersion != "2024-11-20" || input.SKU != "Standard" || input.Capacity != 10 || input.VersionUpgradeOption == nil || *input.VersionUpgradeOption != currentPolicy {
		t.Fatalf("current values were not preserved: %#v", input)
	}
}

func TestDeploymentOperationPropagatesAzureErrorClassification(t *testing.T) {
	azureClient := &recordingDeploymentAzure{
		fakeAzure: newFakeAzure(),
		err:       &azure.OperationError{Class: azure.ErrorUnknown, Operation: "Create or update Model Deployment", Scope: "rg/account/new-deployment", Code: "TooManyRequests", Retryable: true, Message: "retry later"},
	}
	result := NewService(azureClient, &fakeOpenCodex{}, &fakeSettings{}).CreateDeployment(context.Background(), DeploymentRequest{
		TenantID: "tenant", SubscriptionID: "subscription", ResourceGroup: "rg", ResourceName: "account",
		DeploymentName: "new-deployment", ModelName: "gpt-4o", ModelFormat: "OpenAI", ModelVersion: "2024-11-20", SKU: "Standard", Capacity: 10, Confirm: true,
	})
	if result.OK || result.Error == nil || result.Error.Code != "TooManyRequests" || !result.Error.Retryable || result.Error.Scope != "rg/account/new-deployment" {
		t.Fatalf("classified deployment result = %#v", result)
	}
}
