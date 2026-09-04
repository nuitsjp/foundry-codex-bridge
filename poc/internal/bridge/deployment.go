package bridge

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/nuitsjp/foundry-codex-bridge/internal/azure"
)

const (
	versionUpgradeOnceNewDefault = "OnceNewDefaultVersionAvailable"
	versionUpgradeOnceExpired    = "OnceCurrentVersionExpired"
	versionUpgradeNoAuto         = "NoAutoUpgrade"
)

func (s *Service) CreateDeployment(ctx context.Context, request DeploymentRequest) DeploymentOperationResult {
	return s.mutateDeployment(ctx, request, "create")
}

func (s *Service) UpdateDeployment(ctx context.Context, request DeploymentRequest) DeploymentOperationResult {
	return s.mutateDeployment(ctx, request, "update")
}

func (s *Service) mutateDeployment(ctx context.Context, request DeploymentRequest, operation string) DeploymentOperationResult {
	ctx = s.context(ctx)
	result := DeploymentOperationResult{Operation: operation, Status: "running"}
	if !s.mu.TryLock() {
		return deploymentFailure(result, request, "another modifying operation is already running", nil)
	}
	defer s.mu.Unlock()

	if err := validateDeploymentRequest(request, operation); err != nil {
		return deploymentFailure(result, request, err.Error(), nil)
	}
	writer, ok := s.azure.(azure.DeploymentWriter)
	if !ok {
		return deploymentFailure(result, request, "Azure deployment management is unavailable", errors.New("Azure deployment management is unavailable"))
	}

	input := deploymentInput(request)
	if operation == "update" {
		deployments, err := s.azure.Deployments(ctx, request.TenantID, request.SubscriptionID, request.ResourceGroup, request.ResourceName)
		if err != nil {
			return deploymentFailure(result, request, "could not read the current Azure deployment", err)
		}
		current, found := deploymentByName(deployments, request.DeploymentName)
		if !found {
			return deploymentFailure(result, request, "the selected Azure deployment was not found", nil)
		}
		input = mergeDeploymentInput(input, current)
	}

	deployment, err := writer.CreateOrUpdateDeployment(ctx, request.TenantID, request.SubscriptionID, request.ResourceGroup, request.ResourceName, input)
	if err != nil {
		return deploymentFailure(result, request, "Azure deployment operation failed", err)
	}
	result.OK = true
	result.Status = "succeeded"
	result.Deployment = &deployment
	result.Message = "Azure deployment operation completed"
	result.SyncRequired = true
	return result
}

func validateDeploymentRequest(request DeploymentRequest, operation string) error {
	for name, value := range map[string]string{
		"tenant":               request.TenantID,
		"subscription":         request.SubscriptionID,
		"resource group":       request.ResourceGroup,
		"Azure Model Resource": request.ResourceName,
		"deployment name":      request.DeploymentName,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must be selected", name)
		}
	}
	if !request.Confirm {
		return errors.New("confirm the current and changed Azure deployment values before applying")
	}
	if strings.TrimSpace(request.ModelName) == "" && operation == "create" {
		return errors.New("model must be selected")
	}
	if strings.TrimSpace(request.ModelFormat) == "" && operation == "create" {
		return errors.New("model format must be selected")
	}
	if strings.TrimSpace(request.ModelVersion) == "" && operation == "create" {
		return errors.New("model version must be selected")
	}
	if strings.TrimSpace(request.SKU) == "" && operation == "create" {
		return errors.New("SKU must be selected")
	}
	if option := strings.TrimSpace(request.VersionUpgradeOption); option != "" && !validVersionUpgradeOption(option) {
		return fmt.Errorf("version upgrade option %q is not supported", option)
	}
	return nil
}

func validVersionUpgradeOption(value string) bool {
	switch value {
	case versionUpgradeOnceNewDefault, versionUpgradeOnceExpired, versionUpgradeNoAuto:
		return true
	default:
		return false
	}
}

func deploymentInput(request DeploymentRequest) azure.DeploymentInput {
	var policy *string
	if value := strings.TrimSpace(request.VersionUpgradeOption); value != "" {
		policy = &value
	}
	return azure.DeploymentInput{
		Name:                 strings.TrimSpace(request.DeploymentName),
		ModelName:            strings.TrimSpace(request.ModelName),
		ModelFormat:          strings.TrimSpace(request.ModelFormat),
		ModelVersion:         strings.TrimSpace(request.ModelVersion),
		SKU:                  strings.TrimSpace(request.SKU),
		Capacity:             request.Capacity,
		VersionUpgradeOption: policy,
	}
}

func mergeDeploymentInput(input azure.DeploymentInput, current Deployment) azure.DeploymentInput {
	modelName := input.ModelName
	if modelName == "" {
		modelName = current.ModelName
	}
	modelVersion := input.ModelVersion
	if modelVersion == "" {
		modelVersion = current.ModelVersion
	}
	modelFormat := input.ModelFormat
	if modelFormat == "" {
		modelFormat = current.ModelFormat
	}
	sku := input.SKU
	if sku == "" {
		sku = current.SKU
	}
	capacity := input.Capacity
	if capacity == 0 {
		capacity = current.Capacity
	}
	policy := input.VersionUpgradeOption
	if policy == nil && current.VersionUpgradeOption != "" {
		policyValue := current.VersionUpgradeOption
		policy = &policyValue
	}
	return azure.DeploymentInput{
		Name:                 input.Name,
		ModelName:            modelName,
		ModelFormat:          modelFormat,
		ModelVersion:         modelVersion,
		SKU:                  sku,
		Capacity:             capacity,
		VersionUpgradeOption: policy,
	}
}

func deploymentByName(values []Deployment, name string) (Deployment, bool) {
	name = strings.TrimSpace(name)
	for _, value := range values {
		if value.Name == name {
			return value, true
		}
	}
	return Deployment{}, false
}

func deploymentFailure(result DeploymentOperationResult, request DeploymentRequest, message string, cause error) DeploymentOperationResult {
	result.OK = false
	result.Status = "failed"
	result.Message = message
	operation := "Create Azure Model Deployment"
	if result.Operation == "update" {
		operation = "Update Azure Model Deployment"
	}
	scope := strings.Trim(strings.Join([]string{request.ResourceGroup, request.ResourceName, request.DeploymentName}, "/"), "/")
	if cause == nil {
		class := azure.ErrorValidation
		if strings.Contains(strings.ToLower(message), "not found") {
			class = azure.ErrorNotFound
		}
		result.Error = &OperationError{Class: class, Operation: operation, Scope: scope, Message: message}
		return result
	}
	var classified *OperationError
	if errors.As(cause, &classified) {
		result.Error = classified
		if result.Error.Operation == "" {
			result.Error.Operation = operation
		}
		if result.Error.Scope == "" {
			result.Error.Scope = scope
		}
		return result
	}
	result.Error = classifyDeploymentError(cause, operation, scope)
	return result
}

func classifyDeploymentError(err error, operation, scope string) *OperationError {
	return azure.ClassifyError(err, operation, scope)
}
