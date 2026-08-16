package opencodex

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type phase2Call struct {
	args []string
}

type phase2Runner struct {
	responses map[string]CommandResult
	sequences map[string][]CommandResult
	calls     []phase2Call
}

func (r *phase2Runner) Run(_ context.Context, _ string, args ...string) (CommandResult, error) {
	r.calls = append(r.calls, phase2Call{args: append([]string(nil), args...)})
	key := commandKey(args)
	if sequence := r.sequences[key]; len(sequence) > 0 {
		result := sequence[0]
		r.sequences[key] = sequence[1:]
		return result, nil
	}
	result, ok := r.responses[key]
	if !ok {
		return CommandResult{}, errors.New("unexpected command")
	}
	return result, nil
}

func commandKey(args []string) string {
	key := ""
	for _, arg := range args {
		key += "\x00" + arg
	}
	return key
}

func TestPhase2ParsesProviderAndModelResponses(t *testing.T) {
	runner := &phase2Runner{responses: map[string]CommandResult{
		commandKey([]string{"provider", "list", "--json"}):         {Stdout: `{"configured":[{"name":"p1","adapter":"azure-openai","baseUrl":"https://example","defaultModel":"gpt-4o","isDefault":true}]}`},
		commandKey([]string{"models", "list-custom", "--json"}):    {Stdout: `[{"provider":"p1","modelId":"m1"}]`},
		commandKey([]string{"models", "selected", "p1", "--json"}): {Stdout: `{"selected":["m1"]}`},
	}}
	manager := NewManagerWithRunner("", runner, nil)

	providers, err := manager.Providers(context.Background())
	if err != nil || len(providers) != 1 || providers[0].BaseURL != "https://example" {
		t.Fatalf("Providers() = %#v, %v", providers, err)
	}
	custom, err := manager.CustomModels(context.Background())
	if err != nil || !reflect.DeepEqual(custom, []CustomModel{{Provider: "p1", ModelID: "m1"}}) {
		t.Fatalf("CustomModels() = %#v, %v", custom, err)
	}
	selected, err := manager.SelectedModels(context.Background(), "p1")
	if err != nil || !reflect.DeepEqual(selected, []string{"m1"}) {
		t.Fatalf("SelectedModels() = %#v, %v", selected, err)
	}
}

func TestPhase2EnsureSelectedModelsSkipsNoopAndClears(t *testing.T) {
	runner := &phase2Runner{responses: map[string]CommandResult{
		commandKey([]string{"models", "selected", "p1", "--clear", "--json"}): {},
	}, sequences: map[string][]CommandResult{
		commandKey([]string{"models", "selected", "p1", "--json"}): {
			{Stdout: `{"selected":[" b ","a","a"]}`},
			{Stdout: `{"selected":["a"]}`},
		},
	}}
	manager := NewManagerWithRunner("", runner, nil)
	if err := manager.EnsureSelectedModels(context.Background(), "p1", []string{"b", " a "}); err != nil {
		t.Fatal(err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("no-op calls = %#v", runner.calls)
	}
	if err := manager.EnsureSelectedModels(context.Background(), "p1", []string{" ", " "}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(runner.calls[2].args, []string{"models", "selected", "p1", "--clear", "--json"}) {
		t.Fatalf("clear args = %v", runner.calls[2].args)
	}
}

func TestPhase2EnsureCustomModelsIsDeterministicAndKeepsOtherProviders(t *testing.T) {
	runner := &phase2Runner{responses: map[string]CommandResult{
		commandKey([]string{"models", "list-custom", "--json"}):                      {Stdout: `[{"provider":"p1","modelId":"old"},{"provider":"p1","modelId":"keep"},{"provider":"p2","modelId":"untouched"}]`},
		commandKey([]string{"models", "add", "p1", "new-a", "--modalities", "text"}): {},
		commandKey([]string{"models", "remove", "p1/old", "--yes"}):                  {},
	}}
	manager := NewManagerWithRunner("", runner, nil)
	if err := manager.EnsureCustomModels(context.Background(), "p1", []string{" keep ", "new-a", "new-a"}); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"models", "list-custom", "--json"},
		{"models", "add", "p1", "new-a", "--modalities", "text"},
		{"models", "remove", "p1/old", "--yes"},
	}
	if len(runner.calls) != len(want) {
		t.Fatalf("calls = %#v", runner.calls)
	}
	for i := range want {
		if !reflect.DeepEqual(runner.calls[i].args, want[i]) {
			t.Fatalf("call %d = %v, want %v", i, runner.calls[i].args, want[i])
		}
	}
}

func TestPhase2ReturnsErrorsForCommandFailureAndInvalidJSON(t *testing.T) {
	for name, result := range map[string]CommandResult{
		"nonzero":      {Code: 1, Stderr: "failed"},
		"invalid json": {Stdout: "{"},
	} {
		t.Run(name, func(t *testing.T) {
			runner := &phase2Runner{responses: map[string]CommandResult{
				commandKey([]string{"provider", "list", "--json"}): result,
			}}
			_, err := NewManagerWithRunner("", runner, nil).Providers(context.Background())
			if err == nil {
				t.Fatal("Providers() error = nil")
			}
		})
	}
}
