export type AuthState = {
  cliInstalled: boolean;
  cliVersion: string;
  signedIn: boolean;
  username: string;
  tenantId: string;
  message: string;
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
  Tenants(): Promise<Tenant[]>;
  Subscriptions(tenantId: string): Promise<Subscription[]>;
  ResourceGroups(tenantId: string, subscriptionId: string): Promise<ResourceGroup[]>;
  ModelResources(tenantId: string, subscriptionId: string, resourceGroup: string): Promise<ModelResource[]>;
  Deployments(tenantId: string, subscriptionId: string, resourceGroup: string, resourceName: string): Promise<Deployment[]>;
  Models(tenantId: string, subscriptionId: string, resourceGroup: string, resourceName: string): Promise<DeployableModel[]>;
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

export function normalizeList<T>(value: T[] | null | undefined): T[] {
  return value ?? [];
}

export const api = {
  snapshot: () => boundApp().Snapshot(),
  signIn: () => boundApp().SignIn(),
  tenants: () => boundApp().Tenants().then(normalizeList),
  subscriptions: (tenantId: string) => boundApp().Subscriptions(tenantId).then(normalizeList),
  resourceGroups: (tenantId: string, subscriptionId: string) => boundApp().ResourceGroups(tenantId, subscriptionId).then(normalizeList),
  resources: (tenantId: string, subscriptionId: string, resourceGroup: string) => boundApp().ModelResources(tenantId, subscriptionId, resourceGroup).then(normalizeList),
  deployments: (tenantId: string, subscriptionId: string, resourceGroup: string, resourceName: string) => boundApp().Deployments(tenantId, subscriptionId, resourceGroup, resourceName).then(normalizeList),
  models: (tenantId: string, subscriptionId: string, resourceGroup: string, resourceName: string) => boundApp().Models(tenantId, subscriptionId, resourceGroup, resourceName).then(normalizeList),
  prepareOpenCodex: () => boundApp().PrepareOpenCodex(),
  openCodexState: () => boundApp().OpenCodexState(),
  sync: (request: Parameters<BoundApp["Sync"]>[0]) => boundApp().Sync(request),
};
