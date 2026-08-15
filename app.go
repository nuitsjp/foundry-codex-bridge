package main

import (
	"context"
	"os"
	"strings"

	"github.com/nuitsjp/foundry-codex-bridge/internal/bridge"
	"github.com/nuitsjp/foundry-codex-bridge/internal/opencodex"
	"github.com/nuitsjp/foundry-codex-bridge/internal/platform"
)

const azureClientIDEnvironment = "FOUNDRYCODEX_AZURE_CLIENT_ID"

// embeddedAzureClientID is populated by the release build. Development builds
// can override it with FOUNDRYCODEX_AZURE_CLIENT_ID.
var embeddedAzureClientID string

type App struct {
	ctx     context.Context
	service *bridge.Service
}

func NewApp() *App {
	dataDir := platform.AppDataDir()
	clientID := embeddedAzureClientID
	if override := strings.TrimSpace(os.Getenv(azureClientIDEnvironment)); override != "" {
		clientID = override
	}
	azureClient := bridge.NewAzureClient(dataDir, clientID)
	openCodeX := opencodex.NewManager(dataDir)
	return &App{
		service: bridge.NewService(azureClient, openCodeX, bridge.NewFileSettingsStore(dataDir)),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Snapshot() bridge.Snapshot {
	return a.service.Snapshot(a.ctx)
}

func (a *App) SignIn() (bridge.AuthState, error) {
	return a.service.SignIn(a.ctx)
}

func (a *App) SignOut() error {
	return a.service.SignOut(a.ctx)
}

func (a *App) Tenants() ([]bridge.Tenant, error) {
	return a.service.Tenants(a.ctx)
}

func (a *App) Subscriptions(tenantID string) ([]bridge.Subscription, error) {
	return a.service.Subscriptions(a.ctx, tenantID)
}

func (a *App) ResourceGroups(subscriptionID string) ([]bridge.ResourceGroup, error) {
	return a.service.ResourceGroups(a.ctx, subscriptionID)
}

func (a *App) ModelResources(subscriptionID, resourceGroup string) ([]bridge.ModelResource, error) {
	return a.service.ModelResources(a.ctx, subscriptionID, resourceGroup)
}

func (a *App) Deployments(subscriptionID, resourceGroup, resourceName string) ([]bridge.Deployment, error) {
	return a.service.Deployments(a.ctx, subscriptionID, resourceGroup, resourceName)
}

func (a *App) Models(subscriptionID, resourceGroup, resourceName string) ([]bridge.DeployableModel, error) {
	return a.service.Models(a.ctx, subscriptionID, resourceGroup, resourceName)
}

func (a *App) PrepareOpenCodex() (bridge.OpenCodexState, error) {
	return a.service.PrepareOpenCodex(a.ctx)
}

func (a *App) OpenCodexState() (bridge.OpenCodexState, error) {
	return a.service.OpenCodexState(a.ctx)
}

func (a *App) Sync(request bridge.SyncRequest) bridge.SyncResult {
	return a.service.Sync(a.ctx, request)
}
