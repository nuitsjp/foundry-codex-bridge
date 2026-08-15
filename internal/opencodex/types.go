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
	ProviderExists(context.Context, string) (bool, error)
	InstallService(context.Context) error
	EnsureProvider(context.Context, string, string, string) error
	AddPrimaryKey(context.Context, string, string) error
	EnsureCustomModel(context.Context, string, string) error
	SelectModel(context.Context, string, string) error
	SyncCatalog(context.Context) error
	TestResponse(context.Context, int, string, string) error
}
