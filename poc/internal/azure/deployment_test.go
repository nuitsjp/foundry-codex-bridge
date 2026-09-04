package azure

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
)

type testTokenCredential struct{}

func (testTokenCredential) GetToken(context.Context, policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "test-token", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) Do(request *http.Request) (*http.Response, error) {
	return f(request)
}

func newSDKTestClient(transport testTransport) *SDKClient {
	return &SDKClient{
		authState: AuthState{CLIInstalled: true, SignedIn: true},
		credentialFactory: func(string, string) (azcore.TokenCredential, error) {
			return testTokenCredential{}, nil
		},
		clientOptions: &arm.ClientOptions{
			ClientOptions: policy.ClientOptions{
				Transport: transport,
				Retry:     policy.RetryOptions{MaxRetries: -1},
			},
		},
	}
}

func responseFor(request *http.Request, status int, headers http.Header, body string) (*http.Response, error) {
	return &http.Response{
		StatusCode: status,
		Header:     headers,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}, nil
}

func TestModelsMapsAccountModelCapabilitiesAndCapacity(t *testing.T) {
	client := newSDKTestClient(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || !strings.HasSuffix(request.URL.Path, "/accounts/account-1/models") {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL)
		}
		return responseFor(request, http.StatusOK, nil, `{
  "value": [
    {
      "name": "new-responses-model",
      "format": "OpenAI",
      "version": "2025-01-01",
      "capabilities": {"responses": "true", "agentsV2": "false"},
      "maxCapacity": 42,
      "skus": [
        {"name": "Standard", "usageName": "OpenAI.Standard", "rateLimits": [{"count": 1, "renewalPeriod": 60}, {"count": 1000, "renewalPeriod": 60}], "capacity": {"minimum": 1, "maximum": 100, "step": 1, "default": 10, "allowedValues": [1, 10, 100]}},
        {"name": "ProvisionedManaged", "usageName": "OpenAI.ProvisionedManaged", "capacity": {"minimum": 1, "maximum": 42, "step": 1, "default": 4}}
      ]
    },
    {"name": "partner-model", "format": "Meta", "version": "v1", "capabilities": {"agentsV2": "true"}},
    {"name": "agents-only-openai", "format": "OpenAI", "capabilities": {"agentsV2": "true"}}
  ]
}`)
	})

	models, err := client.Models(context.Background(), "tenant-1", "subscription-1", "rg-1", "account-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 3 {
		t.Fatalf("Models() returned %d models, want 3", len(models))
	}

	got := models[0]
	if got.Name != "new-responses-model" || !got.CodexCandidate || got.MaxCapacity != 42 {
		t.Fatalf("model mapping = %#v", got)
	}
	if got.Capabilities["responses"] != "true" {
		t.Fatalf("capabilities = %#v", got.Capabilities)
	}
	if len(got.SKUs) != 2 {
		t.Fatalf("SKUs = %#v", got.SKUs)
	}
	capacity := got.SKUs[0].Capacity
	if got.SKUs[0].Unit != "TPM" || got.SKUs[0].TPMPerCapacityUnit != 1000 || capacity.Minimum != 1 || capacity.Maximum != 100 || capacity.Step != 1 || capacity.Default != 10 || !slices.Equal(capacity.AllowedValues, []int32{1, 10, 100}) {
		t.Fatalf("standard SKU = %#v", got.SKUs[0])
	}
	if got.SKUs[1].Unit != "PTU" || got.SKUs[1].TPMPerCapacityUnit != 1 || got.SKUs[1].Capacity.Maximum != 42 {
		t.Fatalf("provisioned SKU = %#v", got.SKUs[1])
	}
	if !models[1].CodexCandidate || models[2].CodexCandidate {
		t.Fatalf("format-specific candidate mapping = %#v", models)
	}
}

func TestCreateOrUpdateDeploymentMapsRequestAndPollsToCompletion(t *testing.T) {
	var putBody map[string]any
	var requestCount int
	client := newSDKTestClient(func(request *http.Request) (*http.Response, error) {
		requestCount++
		switch {
		case request.Method == http.MethodPut && strings.HasSuffix(request.URL.Path, "/deployments/codex-deployment"):
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(body, &putBody); err != nil {
				t.Fatalf("request body is not JSON: %v", err)
			}
			if _, ok := putBody["name"]; ok {
				t.Fatalf("deployment name must be represented by the request path, body = %s", body)
			}
			return responseFor(request, http.StatusCreated, http.Header{"Location": []string{"https://management.azure.com/poll/codex-deployment"}}, "")
		case request.Method == http.MethodGet && request.URL.Path == "/poll/codex-deployment":
			return responseFor(request, http.StatusOK, nil, `{
  "id": "/subscriptions/subscription-1/resourceGroups/rg-1/providers/Microsoft.CognitiveServices/accounts/account-1/deployments/codex-deployment",
  "name": "codex-deployment",
  "properties": {
    "model": {"name": "new-responses-model", "format": "OpenAI", "version": "2025-01-01"},
    "provisioningState": "Succeeded",
    "versionUpgradeOption": "OnceNewDefaultVersionAvailable"
  },
  "sku": {"name": "Standard", "capacity": 10}
}`)
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL)
			return nil, nil
		}
	})

	policy := "OnceNewDefaultVersionAvailable"
	deployment, err := client.CreateOrUpdateDeployment(context.Background(), "tenant-1", "subscription-1", "rg-1", "account-1", DeploymentInput{
		Name:                 "codex-deployment",
		ModelName:            "new-responses-model",
		ModelFormat:          "OpenAI",
		ModelVersion:         "2025-01-01",
		SKU:                  "Standard",
		Capacity:             10,
		VersionUpgradeOption: &policy,
	})
	if err != nil {
		t.Fatal(err)
	}
	if requestCount != 2 {
		t.Fatalf("request count = %d, want initial PUT and LRO poll", requestCount)
	}
	properties := putBody["properties"].(map[string]any)
	model := properties["model"].(map[string]any)
	if model["name"] != "new-responses-model" || model["format"] != "OpenAI" || model["version"] != "2025-01-01" {
		t.Fatalf("model request mapping = %#v", model)
	}
	if properties["versionUpgradeOption"] != policy {
		t.Fatalf("version upgrade policy mapping = %#v", properties["versionUpgradeOption"])
	}
	sku := putBody["sku"].(map[string]any)
	if sku["name"] != "Standard" || sku["capacity"] != float64(10) {
		t.Fatalf("SKU request mapping = %#v", sku)
	}
	if deployment.Name != "codex-deployment" || deployment.ModelName != "new-responses-model" || deployment.ModelVersion != "2025-01-01" || deployment.SKU != "Standard" || deployment.Capacity != 10 || deployment.VersionUpgradeOption != policy || deployment.ProvisioningState != "Succeeded" {
		t.Fatalf("completed deployment = %#v", deployment)
	}
}

func TestCreateOrUpdateDeploymentOmitsNilVersionUpgradeOptionAndPropagatesAzureError(t *testing.T) {
	var putBody map[string]any
	client := newSDKTestClient(func(request *http.Request) (*http.Response, error) {
		switch {
		case request.Method == http.MethodPut:
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(body, &putBody); err != nil {
				t.Fatal(err)
			}
			return responseFor(request, http.StatusCreated, http.Header{"Location": []string{"https://management.azure.com/poll/error"}}, "")
		case request.Method == http.MethodGet && request.URL.Path == "/poll/error":
			return responseFor(request, http.StatusBadRequest, http.Header{"X-Ms-Request-Id": []string{"request-123"}}, `{"error":{"code":"InvalidDeployment","message":"capacity is unavailable for this request"}}`)
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL)
			return nil, nil
		}
	})

	_, err := client.CreateOrUpdateDeployment(context.Background(), "tenant-1", "subscription-1", "rg-1", "account-1", DeploymentInput{
		Name:         "codex-deployment",
		ModelName:    "new-responses-model",
		ModelFormat:  "OpenAI",
		ModelVersion: "2025-01-01",
		SKU:          "Standard",
		Capacity:     10,
	})
	if err == nil {
		t.Fatal("CreateOrUpdateDeployment() error = nil, want Azure error")
	}
	var operationError *OperationError
	if !errors.As(err, &operationError) {
		t.Fatalf("error type = %T, want *OperationError", err)
	}
	if operationError.Code != "InvalidDeployment" || operationError.Scope != "rg-1/account-1/codex-deployment" || operationError.Retryable {
		t.Fatalf("classified error = %#v", operationError)
	}
	if _, ok := putBody["properties"].(map[string]any)["versionUpgradeOption"]; ok {
		t.Fatalf("nil policy must be omitted from SDK request: %#v", putBody)
	}
}

func TestCreateOrUpdateDeploymentDoesNotRetryAzureRequest(t *testing.T) {
	var requestCount int
	client := newSDKTestClient(nil)
	client.clientOptions = &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{Transport: testTransport(func(request *http.Request) (*http.Response, error) {
			requestCount++
			return responseFor(request, http.StatusTooManyRequests, http.Header{"X-Ms-Request-Id": []string{"request-429"}}, `{"error":{"code":"TooManyRequests","message":"try later"}}`)
		})},
	}

	_, err := client.CreateOrUpdateDeployment(context.Background(), "tenant-1", "subscription-1", "rg-1", "account-1", DeploymentInput{
		Name: "codex-deployment", ModelName: "new-responses-model", ModelVersion: "2025-01-01", SKU: "Standard", Capacity: 10,
	})
	if err == nil {
		t.Fatal("CreateOrUpdateDeployment() error = nil, want Azure error")
	}
	if requestCount != 1 {
		t.Fatalf("Azure request count = %d, want one attempt without automatic retry", requestCount)
	}
}
