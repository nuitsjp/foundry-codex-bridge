package opencodex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Manager struct {
	dataDir string
	runner  CommandRunner
	client  *http.Client
}

func NewManager(dataDir string) *Manager {
	return &Manager{
		dataDir: dataDir,
		runner:  NewLocalRunner(dataDir),
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func NewManagerWithRunner(dataDir string, runner CommandRunner, client *http.Client) *Manager {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &Manager{dataDir: dataDir, runner: runner, client: client}
}

func (m *Manager) State(ctx context.Context) (State, error) {
	return m.state(ctx)
}

func (m *Manager) Prepare(ctx context.Context) (State, error) {
	return m.prepare(ctx)
}

func (m *Manager) state(ctx context.Context) (State, error) {
	info := m.nodeInfo(ctx)
	path := filepath.Join(m.dataDir, "opencodex")
	result := State{NodeInfo: info, Path: path}
	health, err := m.health(ctx)
	if err == nil {
		result.Installed = true
		result.Health = health
		result.Message = "opencodex is ready"
		return result, nil
	}
	if _, pathErr := m.runnerPath(); pathErr == nil {
		result.Installed = true
		result.Message = "opencodex is installed but not ready"
		return result, nil
	}
	result.Message = "opencodex is not installed"
	return result, nil
}

func (m *Manager) EnsureService(ctx context.Context) error {
	result, err := m.runner.Run(ctx, "", "status", "--json")
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return commandFailure("ocx status", result)
	}
	var status struct {
		Startup struct {
			ServiceInstalled bool `json:"serviceInstalled"`
			ServiceViable    bool `json:"serviceViable"`
			ServiceConflict  bool `json:"serviceConflict"`
		} `json:"startup"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &status); err != nil {
		return errors.New("ocx status returned invalid JSON")
	}
	if status.Startup.ServiceViable {
		return nil
	}
	command := "install"
	if status.Startup.ServiceInstalled && !status.Startup.ServiceConflict {
		command = "repair"
	}
	result, err = m.runner.Run(ctx, "", "service", command)
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return commandFailure("ocx service "+command, result)
	}
	return nil
}

func (m *Manager) ProviderExists(ctx context.Context, providerID string) (bool, error) {
	result, err := m.runner.Run(ctx, "", "provider", "list", "--json")
	if err != nil {
		return false, err
	}
	if result.Code != 0 {
		return false, commandFailure("ocx provider list", result)
	}
	var payload struct {
		Configured []struct {
			Name string `json:"name"`
		} `json:"configured"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &payload); err != nil {
		return false, errors.New("ocx provider list returned invalid JSON")
	}
	for _, provider := range payload.Configured {
		if provider.Name == providerID {
			return true, nil
		}
	}
	return false, nil
}

func (m *Manager) EnsureProvider(ctx context.Context, providerID, baseURL, model string) error {
	result, err := m.runner.Run(ctx, "", "provider", "add", providerID,
		"--adapter", "azure-openai",
		"--base-url", baseURL,
		"--default-model", model,
		"--force", "--json")
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return commandFailure("ocx provider add", result)
	}
	result, err = m.runner.Run(ctx, "", "provider", "edit", providerID, "--live-models", "off", "--json")
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return commandFailure("ocx provider edit", result)
	}
	return nil
}

func (m *Manager) AddPrimaryKey(ctx context.Context, providerID, primaryKey string) error {
	if strings.TrimSpace(primaryKey) == "" {
		return errors.New("Azure returned an empty PrimaryKey")
	}
	result, err := m.runner.Run(ctx, primaryKey+"\n", "account", "add-key", providerID, "--label", "primary", "--json")
	if err != nil {
		return redactError(err, primaryKey, "")
	}
	if result.Code != 0 {
		return commandFailureWithSecret("ocx account add-key", result, primaryKey)
	}
	return nil
}

func (m *Manager) EnsureCustomModel(ctx context.Context, providerID, model string) error {
	result, err := m.runner.Run(ctx, "", "models", "list-custom", "--json")
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return commandFailure("ocx models list-custom", result)
	}
	var entries []struct {
		Provider string `json:"provider"`
		ModelID  string `json:"modelId"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &entries); err != nil {
		return fmt.Errorf("ocx models list-custom returned invalid JSON")
	}
	for _, entry := range entries {
		if entry.Provider == providerID && entry.ModelID == model {
			return nil
		}
	}
	result, err = m.runner.Run(ctx, "", "models", "add", providerID, model, "--modalities", "text")
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return commandFailure("ocx models add", result)
	}
	return nil
}

func (m *Manager) SelectModel(ctx context.Context, providerID, model string) error {
	result, err := m.runner.Run(ctx, "", "models", "selected", providerID, "--set", model, "--json")
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return commandFailure("ocx models selected", result)
	}
	return nil
}

func (m *Manager) SyncCatalog(ctx context.Context) error {
	result, err := m.runner.Run(ctx, "", "sync")
	if err != nil {
		return err
	}
	if result.Code != 0 {
		return commandFailure("ocx sync", result)
	}
	return nil
}

func (m *Manager) TestResponse(ctx context.Context, port int, providerID, model string) error {
	if port <= 0 {
		return errors.New("opencodex did not report a valid port")
	}
	payload := map[string]any{
		"model":             providerID + "/" + model,
		"input":             "Reply with OK.",
		"stream":            false,
		"store":             false,
		"max_output_tokens": 16,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("http://127.0.0.1:%d/v1/responses", port), strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := m.client.Do(request)
	if err != nil {
		return fmt.Errorf("Responses endpoint request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Responses endpoint returned HTTP %d", response.StatusCode)
	}
	var responseBody map[string]any
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		return errors.New("Responses endpoint returned an invalid JSON response")
	}
	if len(responseBody) == 0 {
		return errors.New("Responses endpoint returned an empty response")
	}
	return nil
}

func (m *Manager) health(ctx context.Context) (Health, error) {
	result, err := m.runner.Run(ctx, "", "health", "--json")
	if err != nil {
		return Health{}, err
	}
	var payload struct {
		OK   bool `json:"ok"`
		PID  int  `json:"pid"`
		Port int  `json:"port"`
	}
	if decodeErr := json.Unmarshal([]byte(result.Stdout), &payload); decodeErr != nil {
		return Health{}, errors.New("ocx health returned invalid JSON")
	}
	if result.Code != 0 || !payload.OK {
		return Health{PID: payload.PID, Port: payload.Port}, errors.New("opencodex is not healthy")
	}
	return Health{Ready: true, PID: payload.PID, Port: payload.Port}, nil
}

func (m *Manager) runnerPath() (string, error) {
	if local, ok := m.runner.(*LocalRunner); ok {
		return local.commandPath()
	}
	return "ocx", nil
}

func commandFailure(command string, result CommandResult) error {
	return commandFailureWithSecret(command, result, "")
}

func commandFailureWithSecret(command string, result CommandResult, secret string) error {
	detail := strings.TrimSpace(result.Stderr)
	if detail == "" {
		detail = strings.TrimSpace(result.Stdout)
	}
	if detail == "" {
		detail = "command failed"
	}
	detail = redactSecret(detail, secret)
	return fmt.Errorf("%s: %s", command, detail)
}

func runCommand(ctx context.Context, path, stdin string, args ...string) (CommandResult, error) {
	command := exec.CommandContext(ctx, path, args...)
	if stdin != "" {
		command.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr strings.Builder
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	result := CommandResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		return result, redactError(err, stdin, result.Stderr)
	}
	return result, nil
}

func execLookPath(name string) (string, error) {
	return exec.LookPath(name)
}
