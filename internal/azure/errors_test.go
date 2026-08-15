package azure

import "testing"

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
