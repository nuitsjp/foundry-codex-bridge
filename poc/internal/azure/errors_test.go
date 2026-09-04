package azure

import (
	"net/http"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

func TestClassifyErrorRecognizesAzureCLIAuthenticationFailure(t *testing.T) {
	classified := ClassifyError(&azidentity.AuthenticationFailedError{}, "List resource groups", "subscription")
	if classified.Class != ErrorAuthentication {
		t.Fatalf("ClassifyError() class = %q, want %q", classified.Class, ErrorAuthentication)
	}
	if classified.Message != "Azure sign-in is no longer valid. Sign in again." {
		t.Fatalf("ClassifyError() message = %q", classified.Message)
	}
}

func TestClassifyErrorMarksTransientAzureResponsesRetryable(t *testing.T) {
	classified := ClassifyError(&azcore.ResponseError{
		StatusCode: 429,
		ErrorCode:  "TooManyRequests",
		RawResponse: &http.Response{Header: http.Header{
			"X-Ms-Request-Id": []string{"request-429"},
		}},
	}, "Create deployment", "rg/account/deployment")
	if !classified.Retryable || classified.Code != "TooManyRequests" || classified.Scope != "rg/account/deployment" || classified.RequestID != "request-429" {
		t.Fatalf("classified = %#v", classified)
	}
}

func TestCodexCandidateUsesFormatSpecificCapability(t *testing.T) {
	tests := []struct {
		name         string
		format       string
		capabilities map[string]string
		want         bool
	}{
		{name: "openai responses", format: "OpenAI", capabilities: map[string]string{"responses": "true"}, want: true},
		{name: "openai agents only", format: "OpenAI", capabilities: map[string]string{"agentsV2": "true"}, want: false},
		{name: "partner agents", format: "Meta", capabilities: map[string]string{"agentsV2": "true"}, want: true},
		{name: "false value", format: "OpenAI", capabilities: map[string]string{"responses": "false"}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := CodexCandidate(test.format, test.capabilities); got != test.want {
				t.Fatalf("CodexCandidate() = %v, want %v", got, test.want)
			}
		})
	}
}
