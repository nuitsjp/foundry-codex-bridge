package azure

import (
	"testing"

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
