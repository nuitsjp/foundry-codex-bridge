package opencodex

import "context"

type CommandResult struct {
	Stdout string
	Stderr string
	Code   int
}

type CommandRunner interface {
	Run(context.Context, string, ...string) (CommandResult, error)
}

type Provider struct {
	Name         string `json:"name"`
	Adapter      string `json:"adapter"`
	BaseURL      string `json:"baseUrl"`
	DefaultModel string `json:"defaultModel"`
	IsDefault    bool   `json:"isDefault"`
	LiveModels   bool   `json:"liveModels"`
}

type CustomModel struct {
	Provider string `json:"provider"`
	ModelID  string `json:"modelId"`
}

type ComboTarget struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type Combo struct {
	ID      string        `json:"id"`
	Targets []ComboTarget `json:"targets"`
}

type NodeInfo struct {
	NodeInstalled bool   `json:"nodeInstalled"`
	NodeVersion   string `json:"nodeVersion"`
	NpmInstalled  bool   `json:"npmInstalled"`
	NpmVersion    string `json:"npmVersion"`
	RequiredNode  string `json:"requiredNode"`
	Compatible    bool   `json:"compatible"`
}

type Health struct {
	Ready bool `json:"ready"`
	PID   int  `json:"pid"`
	Port  int  `json:"port"`
}

type State struct {
	NodeInfo
	Installed bool   `json:"installed"`
	Path      string `json:"path"`
	Health    Health `json:"health"`
	Message   string `json:"message"`
}

type ManagerAPI interface {
	State(context.Context) (State, error)
	Prepare(context.Context) (State, error)
	Providers(context.Context) ([]Provider, error)
	Provider(context.Context, string) (Provider, error)
	CustomModels(context.Context) ([]CustomModel, error)
	SelectedModels(context.Context, string) ([]string, error)
	EnsureSelectedModels(context.Context, string, []string) error
	EnsureCustomModels(context.Context, string, []string) error
	Combos(context.Context) ([]Combo, error)
	SetDefaultProvider(context.Context, string) error
	RemoveProvider(context.Context, string) error
	StartService(context.Context) error
	StopService(context.Context) error
	RepairService(context.Context) error
	SetPort(context.Context, int) error
	UpdateLatest(context.Context) error
	SyncCatalogRestartCodex(context.Context) error
	ProviderExists(context.Context, string) (bool, error)
	EnsureService(context.Context) error
	EnsureProvider(context.Context, string, string, string) error
	AddPrimaryKey(context.Context, string, string) error
	EnsureCustomModel(context.Context, string, string) error
	SelectModel(context.Context, string, string) error
	SyncCatalog(context.Context) error
	TestResponse(context.Context, int, string, string) error
}
