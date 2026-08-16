package main

import (
	"context"

	"github.com/nuitsjp/foundry-codex-bridge/internal/bridge"
	"github.com/nuitsjp/foundry-codex-bridge/internal/opencodex"
	"github.com/nuitsjp/foundry-codex-bridge/internal/platform"
)

type App struct {
	ctx     context.Context
	service *bridge.Service
}

func NewApp() *App {
	dataDir := platform.AppDataDir()
	azureClient := bridge.NewAzureClient()
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

func (a *App) Tenants() ([]bridge.Tenant, error) {
	return a.service.Tenants(a.ctx)
}

func (a *App) Subscriptions(tenantID string) ([]bridge.Subscription, error) {
	return a.service.Subscriptions(a.ctx, tenantID)
}

func (a *App) ResourceGroups(tenantID, subscriptionID string) ([]bridge.ResourceGroup, error) {
	return a.service.ResourceGroups(a.ctx, tenantID, subscriptionID)
}

func (a *App) ModelResources(tenantID, subscriptionID, resourceGroup string) ([]bridge.ModelResource, error) {
	return a.service.ModelResources(a.ctx, tenantID, subscriptionID, resourceGroup)
}

func (a *App) Deployments(tenantID, subscriptionID, resourceGroup, resourceName string) ([]bridge.Deployment, error) {
	return a.service.Deployments(a.ctx, tenantID, subscriptionID, resourceGroup, resourceName)
}

func (a *App) Models(tenantID, subscriptionID, resourceGroup, resourceName string) ([]bridge.DeployableModel, error) {
	return a.service.Models(a.ctx, tenantID, subscriptionID, resourceGroup, resourceName)
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
