export type AuthState = {
  signedIn: boolean;
  username: string;
  tenantId: string;
};

export type Tenant = { id: string; displayName: string };
export type Subscription = { id: string; name: string; tenantId: string; state: string };
export type ResourceGroup = { name: string; location: string };
export type ModelResource = {
  id: string;
  name: string;
  kind: string;
  location: string;
  endpoint: string;
  disableLocalAuth: boolean;
};
export type Deployment = {
  id: string;
  name: string;
  modelName: string;
  modelFormat: string;
  modelVersion: string;
  sku: string;
  capacity: number;
  provisioningState: string;
};
export type DeployableModel = {
  name: string;
  format: string;
  version: string;
  capabilities: Record<string, string>;
  codexCandidate: boolean;
};
export type OpenCodexState = {
  nodeInstalled: boolean;
  nodeVersion: string;
  npmInstalled: boolean;
  npmVersion: string;
  requiredNode: string;
  compatible: boolean;
  installed: boolean;
  path: string;
  health: { ready: boolean; pid: number; port: number };
  message: string;
};
export type Selection = {
  tenantId: string;
  subscriptionId: string;
  resourceGroup: string;
  resourceId: string;
  resourceName: string;
  deploymentName: string;
  providerId: string;
};
export type Snapshot = {
  auth: AuthState;
  selection: Selection;
  openCodex: OpenCodexState;
  azureClientReady: boolean;
};
export type SyncStage = { name: string; status: "succeeded" | "failed"; message: string };
export type SyncResult = { ok: boolean; providerId: string; deployment: string; stages: SyncStage[] };

type BoundApp = {
  Snapshot(): Promise<Snapshot>;
  SignIn(): Promise<AuthState>;
  SignOut(): Promise<void>;
  Tenants(): Promise<Tenant[]>;
  Subscriptions(tenantId: string): Promise<Subscription[]>;
  ResourceGroups(subscriptionId: string): Promise<ResourceGroup[]>;
  ModelResources(subscriptionId: string, resourceGroup: string): Promise<ModelResource[]>;
  Deployments(subscriptionId: string, resourceGroup: string, resourceName: string): Promise<Deployment[]>;
  Models(subscriptionId: string, resourceGroup: string, resourceName: string): Promise<DeployableModel[]>;
  PrepareOpenCodex(): Promise<OpenCodexState>;
  OpenCodexState(): Promise<OpenCodexState>;
  Sync(request: {
    tenantId: string;
    subscriptionId: string;
    resourceGroup: string;
    resourceName: string;
    deploymentName: string;
    providerId: string;
    confirmCosts: boolean;
  }): Promise<SyncResult>;
};

declare global {
  interface Window {
    go?: { main?: { App?: BoundApp } };
  }
}

function boundApp(): BoundApp {
  const app = window.go?.main?.App;
  if (!app) {
    throw new Error("Wails runtime is not available. Start this page through the desktop app.");
  }
  return app;
}

export const api = {
  snapshot: () => boundApp().Snapshot(),
  signIn: () => boundApp().SignIn(),
  signOut: () => boundApp().SignOut(),
  tenants: () => boundApp().Tenants(),
  subscriptions: (tenantId: string) => boundApp().Subscriptions(tenantId),
  resourceGroups: (subscriptionId: string) => boundApp().ResourceGroups(subscriptionId),
  resources: (subscriptionId: string, resourceGroup: string) => boundApp().ModelResources(subscriptionId, resourceGroup),
  deployments: (subscriptionId: string, resourceGroup: string, resourceName: string) => boundApp().Deployments(subscriptionId, resourceGroup, resourceName),
  models: (subscriptionId: string, resourceGroup: string, resourceName: string) => boundApp().Models(subscriptionId, resourceGroup, resourceName),
  prepareOpenCodex: () => boundApp().PrepareOpenCodex(),
  openCodexState: () => boundApp().OpenCodexState(),
  sync: (request: Parameters<BoundApp["Sync"]>[0]) => boundApp().Sync(request),
};
