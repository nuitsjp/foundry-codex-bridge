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
  versionUpgradeOption: string;
};
export type CapacityConstraints = {
  minimum: number;
  maximum: number;
  step: number;
  default: number;
  allowedValues: number[];
};
export type ModelSKU = {
  name: string;
  usageName: string;
  capacity: CapacityConstraints;
  unit: string;
  tpmPerCapacityUnit: number;
};
export type DeployableModel = {
  name: string;
  format: string;
  version: string;
  capabilities: Record<string, string>;
  codexCandidate: boolean;
  maxCapacity: number;
  skus: ModelSKU[];
};
export type DeploymentRequest = {
  tenantId: string;
  subscriptionId: string;
  resourceGroup: string;
  resourceName: string;
  deploymentName: string;
  modelName: string;
  modelFormat: string;
  modelVersion: string;
  sku: string;
  capacity: number;
  versionUpgradeOption?: string;
  confirm: boolean;
};
export type DeploymentOperationError = {
  class: string;
  operation: string;
  scope: string;
  code: string;
  requestId: string;
  message: string;
  retryable: boolean;
};
export type DeploymentOperationResult = {
  ok: boolean;
  operation: "create" | "update";
  status: "running" | "succeeded" | "failed";
  deployment?: Deployment;
  error?: DeploymentOperationError;
  message: string;
  syncRequired: boolean;
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
  deploymentNames: string[];
  defaultDeploymentName: string;
  providerId: string;
};
export type Snapshot = {
  auth: AuthState;
  selection: Selection;
  openCodex: OpenCodexState;
  azureClientReady: boolean;
};
export type SyncStage = { name: string; status: "pending" | "succeeded" | "failed"; message: string };
export type ConnectionResult = { deployment: string; status: string; message: string };
export type SyncResult = {
  ok: boolean;
  providerId: string;
  deployment: string;
  stages: SyncStage[];
  connections: ConnectionResult[];
};
export type SyncRequest = {
  tenantId: string;
  subscriptionId: string;
  resourceGroup: string;
  resourceName: string;
  deploymentNames: string[];
  defaultDeploymentName: string;
  providerId: string;
  confirmCosts: boolean;
};
export type SyncChange = { area: string; action: string; details: string };
export type SyncPreview = { ok: boolean; providerId: string; changes: SyncChange[]; message: string };
export type ManagedProvider = {
  resourceId: string;
  providerId: string;
  tenantId: string;
  subscriptionId: string;
  resourceGroup: string;
  resourceName: string;
  location: string;
  deploymentNames: string[];
  defaultDeploymentName: string;
};
export type DisconnectPreview = {
  ok: boolean;
  managed: ManagedProvider;
  isDefault: boolean;
  dependentCombos: string[];
  replacementProviders: string[];
  message: string;
};
export type DisconnectRequest = { resourceId: string; replacementProviderId: string };
export type ActionResult = { ok: boolean; stages: SyncStage[] };

type BoundApp = {
  Snapshot(): Promise<Snapshot>;
  SignIn(): Promise<AuthState>;
  Tenants(): Promise<Tenant[]>;
  Subscriptions(tenantId: string): Promise<Subscription[]>;
  ResourceGroups(tenantId: string, subscriptionId: string): Promise<ResourceGroup[]>;
  ModelResources(tenantId: string, subscriptionId: string, resourceGroup: string): Promise<ModelResource[]>;
  Deployments(tenantId: string, subscriptionId: string, resourceGroup: string, resourceName: string): Promise<Deployment[]>;
  Models(tenantId: string, subscriptionId: string, resourceGroup: string, resourceName: string): Promise<DeployableModel[]>;
  CreateDeployment(request: DeploymentRequest): Promise<DeploymentOperationResult>;
  UpdateDeployment(request: DeploymentRequest): Promise<DeploymentOperationResult>;
  PrepareOpenCodex(): Promise<OpenCodexState>;
  OpenCodexState(): Promise<OpenCodexState>;
  PreviewSync(request: SyncRequest): Promise<SyncPreview>;
  ManagedProviders(): Promise<ManagedProvider[]>;
  PreviewDisconnect(resourceId: string): Promise<DisconnectPreview>;
  Disconnect(request: DisconnectRequest): Promise<ActionResult>;
  StartOpenCodex(): Promise<ActionResult>;
  StopOpenCodex(): Promise<ActionResult>;
  RepairOpenCodex(): Promise<ActionResult>;
  UpdateOpenCodex(): Promise<ActionResult>;
  ChangeOpenCodexPort(port: number): Promise<ActionResult>;
  RestartCodexCatalog(): Promise<ActionResult>;
  Sync(request: SyncRequest): Promise<SyncResult>;
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

function normalizeSyncResult(value: SyncResult): SyncResult {
  return { ...value, stages: normalizeList(value.stages), connections: normalizeList(value.connections) };
}

function normalizeManagedProvider(value: ManagedProvider): ManagedProvider {
  return {
    ...value,
    deploymentNames: normalizeList(value.deploymentNames),
  };
}

function normalizeSyncPreview(value: SyncPreview): SyncPreview {
  return { ...value, changes: normalizeList(value.changes) };
}

function normalizeDisconnectPreview(value: DisconnectPreview): DisconnectPreview {
  return {
    ...value,
    managed: normalizeManagedProvider(value.managed),
    dependentCombos: normalizeList(value.dependentCombos),
    replacementProviders: normalizeList(value.replacementProviders),
  };
}

function normalizeActionResult(value: ActionResult): ActionResult {
  return { ...value, stages: normalizeList(value.stages) };
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
  createDeployment: (request: DeploymentRequest) => boundApp().CreateDeployment(request),
  updateDeployment: (request: DeploymentRequest) => boundApp().UpdateDeployment(request),
  prepareOpenCodex: () => boundApp().PrepareOpenCodex(),
  openCodexState: () => boundApp().OpenCodexState(),
  previewSync: (request: SyncRequest) => boundApp().PreviewSync(request).then(normalizeSyncPreview),
  managedProviders: () => boundApp().ManagedProviders().then((value) => normalizeList(value).map(normalizeManagedProvider)),
  previewDisconnect: (resourceId: string) => boundApp().PreviewDisconnect(resourceId).then(normalizeDisconnectPreview),
  disconnect: (request: DisconnectRequest) => boundApp().Disconnect(request).then(normalizeActionResult),
  startOpenCodex: () => boundApp().StartOpenCodex().then(normalizeActionResult),
  stopOpenCodex: () => boundApp().StopOpenCodex().then(normalizeActionResult),
  repairOpenCodex: () => boundApp().RepairOpenCodex().then(normalizeActionResult),
  updateOpenCodex: () => boundApp().UpdateOpenCodex().then(normalizeActionResult),
  changeOpenCodexPort: (port: number) => boundApp().ChangeOpenCodexPort(port).then(normalizeActionResult),
  restartCodexCatalog: () => boundApp().RestartCodexCatalog().then(normalizeActionResult),
  sync: (request: Parameters<BoundApp["Sync"]>[0]) => boundApp().Sync(request).then(normalizeSyncResult),
};
