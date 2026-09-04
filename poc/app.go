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

func (a *App) CreateDeployment(request bridge.DeploymentRequest) bridge.DeploymentOperationResult {
	return a.service.CreateDeployment(a.ctx, request)
}

func (a *App) UpdateDeployment(request bridge.DeploymentRequest) bridge.DeploymentOperationResult {
	return a.service.UpdateDeployment(a.ctx, request)
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

func (a *App) PreviewSync(request bridge.SyncRequest) bridge.SyncPreview {
	return a.service.PreviewSync(a.ctx, request)
}

func (a *App) ManagedProviders() ([]bridge.ManagedProvider, error) {
	return a.service.ManagedProviders()
}

func (a *App) PreviewDisconnect(resourceID string) bridge.DisconnectPreview {
	return a.service.PreviewDisconnect(a.ctx, resourceID)
}

func (a *App) Disconnect(request bridge.DisconnectRequest) bridge.ActionResult {
	return a.service.Disconnect(a.ctx, request)
}

func (a *App) StartOpenCodex() bridge.ActionResult {
	return a.service.StartOpenCodex(a.ctx)
}

func (a *App) StopOpenCodex() bridge.ActionResult {
	return a.service.StopOpenCodex(a.ctx)
}

func (a *App) RepairOpenCodex() bridge.ActionResult {
	return a.service.RepairOpenCodex(a.ctx)
}

func (a *App) UpdateOpenCodex() bridge.ActionResult {
	return a.service.UpdateOpenCodex(a.ctx)
}

func (a *App) ChangeOpenCodexPort(port int) bridge.ActionResult {
	return a.service.ChangeOpenCodexPort(a.ctx, port)
}

func (a *App) RestartCodexCatalog() bridge.ActionResult {
	return a.service.RestartCodexCatalog(a.ctx)
}
