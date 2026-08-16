import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import type { Snapshot } from "./api";

const tenants = [
  { id: "tenant-a", displayName: "Tenant A" },
  { id: "tenant-b", displayName: "Tenant B" },
];

function snapshot(savedTenantId: string): Snapshot {
  return {
    auth: {
      cliInstalled: true,
      cliVersion: "2.0.0",
      signedIn: true,
      username: "user@example.com",
      tenantId: "tenant-b",
      message: "",
    },
    selection: {
      tenantId: savedTenantId,
      subscriptionId: "",
      resourceGroup: "",
      resourceId: "",
      resourceName: "",
      deploymentNames: [],
      defaultDeploymentName: "",
      providerId: "",
    },
    openCodex: {
      nodeInstalled: false,
      nodeVersion: "",
      npmInstalled: false,
      npmVersion: "",
      requiredNode: ">=18",
      compatible: false,
      installed: false,
      path: "",
      health: { ready: false, pid: 0, port: 0 },
      message: "",
    },
    azureClientReady: true,
  };
}

function installBoundApp(savedTenantId: string) {
  const app = {
    Snapshot: vi.fn().mockResolvedValue(snapshot(savedTenantId)),
    Tenants: vi.fn().mockResolvedValue(tenants),
    Subscriptions: vi.fn().mockResolvedValue(null),
    ResourceGroups: vi.fn().mockResolvedValue([]),
    ModelResources: vi.fn().mockResolvedValue([]),
    Deployments: vi.fn().mockResolvedValue([]),
    Models: vi.fn().mockResolvedValue([]),
    PrepareOpenCodex: vi.fn().mockResolvedValue(snapshot(savedTenantId).openCodex),
    OpenCodexState: vi.fn().mockResolvedValue(snapshot(savedTenantId).openCodex),
    ManagedProviders: vi.fn().mockResolvedValue([]),
    PreviewSync: vi.fn(),
    PreviewDisconnect: vi.fn(),
    Disconnect: vi.fn(),
    StartOpenCodex: vi.fn(),
    StopOpenCodex: vi.fn(),
    RepairOpenCodex: vi.fn(),
    UpdateOpenCodex: vi.fn(),
    ChangeOpenCodexPort: vi.fn(),
    RestartCodexCatalog: vi.fn(),
    CreateDeployment: vi.fn(),
    UpdateDeployment: vi.fn(),
    Sync: vi.fn(),
  };
  window.go = { main: { App: app as never } };
  return app;
}

const deploymentList = [
  { id: "deployment-a", name: "deployment-a", modelName: "gpt-4o", modelFormat: "OpenAI", modelVersion: "2024-08-06", sku: "Standard", capacity: 10, versionUpgradeOption: "NoAutoUpgrade", provisioningState: "Succeeded" },
  { id: "deployment-b", name: "deployment-b", modelName: "gpt-4.1", modelFormat: "OpenAI", modelVersion: "2025-04-14", sku: "Standard", capacity: 10, versionUpgradeOption: "", provisioningState: "Succeeded" },
];

function installDeploymentApp(savedSelection: Partial<Snapshot["selection"]> = {}, ready = false) {
  const app = installBoundApp("");
  const base = snapshot("");
  app.Snapshot.mockResolvedValue({
    ...base,
    selection: { ...base.selection, ...savedSelection },
    openCodex: ready ? { ...base.openCodex, nodeInstalled: true, npmInstalled: true, compatible: true, installed: true, health: { ready: true, pid: 1, port: 8080 } } : base.openCodex,
  });
  app.Subscriptions.mockResolvedValue([{ id: "subscription-1", name: "Subscription 1", tenantId: "tenant-b", state: "Enabled" }]);
  app.ResourceGroups.mockResolvedValue([{ name: "group-1", location: "japaneast" }]);
  app.ModelResources.mockResolvedValue([{ id: "resource-1", name: "model-resource", kind: "OpenAI", location: "japaneast", endpoint: "https://example.openai.azure.com", disableLocalAuth: false }]);
  app.Deployments.mockResolvedValue(deploymentList);
  app.Models.mockResolvedValue([
    { name: "gpt-4o", format: "OpenAI", version: "2024-08-06", capabilities: { responses: "true" }, codexCandidate: true, maxCapacity: 150, skus: [{ name: "Standard", usageName: "OpenAI.Standard", unit: "TPM", tpmPerCapacityUnit: 1000, capacity: { minimum: 1, maximum: 150, step: 1, default: 10, allowedValues: [1, 10, 100, 150] } }] },
    { name: "gpt-4.1", format: "OpenAI", version: "2025-04-14", capabilities: { responses: "true" }, codexCandidate: true, maxCapacity: 150, skus: [{ name: "Standard", usageName: "OpenAI.Standard", unit: "TPM", tpmPerCapacityUnit: 1000, capacity: { minimum: 1, maximum: 150, step: 1, default: 10, allowedValues: [] } }] },
    { name: "gpt-4o-ptu", format: "OpenAI", version: "2024-08-06", capabilities: { responses: "true" }, codexCandidate: true, maxCapacity: 42, skus: [{ name: "ProvisionedManaged", usageName: "OpenAI.ProvisionedManaged", unit: "PTU", tpmPerCapacityUnit: 1, capacity: { minimum: 1, maximum: 42, step: 1, default: 4, allowedValues: [] } }] },
  ]);
  return app;
}

async function renderAndWaitForInitialLoad(savedTenantId: string) {
  const app = installBoundApp(savedTenantId);
  render(<App />);

  await waitFor(() => expect(app.Subscriptions).toHaveBeenCalledWith(expect.any(String)));
  await waitFor(() => expect(screen.queryByText("初期状態を読み込み中")).not.toBeInTheDocument());
  return app;
}

describe("App initial tenant selection", () => {
  afterEach(() => {
    cleanup();
    window.go = undefined;
  });

  it("uses the Azure CLI tenant and keeps Connect visible when subscriptions is null", async () => {
    const app = await renderAndWaitForInitialLoad("");

    expect(app.Snapshot).toHaveBeenCalledTimes(1);
    expect(app.Tenants).toHaveBeenCalledTimes(1);
    expect(app.Subscriptions).toHaveBeenCalledWith("tenant-b");
    expect(screen.getByRole("heading", { name: "Azureへ接続" })).toBeInTheDocument();
  });

  it("uses the saved tenant when one exists", async () => {
    const app = await renderAndWaitForInitialLoad("tenant-a");

    expect(app.Subscriptions).toHaveBeenCalledWith("tenant-a");
    expect(screen.getByRole("heading", { name: "Azureへ接続" })).toBeInTheDocument();
  });

  it("shows progress in the stable footer status area", async () => {
    const app = installBoundApp("");
    app.Snapshot.mockReturnValue(new Promise(() => undefined));
    render(<App />);

    const progress = await screen.findByText("初期状態を読み込み中");
    const status = screen.getByRole("status");
    expect(status).toContainElement(progress);
    expect(status).toHaveClass("activity-status");
    expect(document.querySelector(".content")).not.toHaveTextContent("初期状態を読み込み中");
    expect(screen.queryByText("Azure CLIをインストールしてからBridgeを再起動してください。")).not.toBeInTheDocument();
    expect(screen.queryByText("Azure CLI未検出")).not.toBeInTheDocument();
  });

  it("does not report Azure CLI as missing when initial state loading fails", async () => {
    const app = installBoundApp("");
    app.Snapshot.mockRejectedValue(new Error("Snapshot failed"));
    render(<App />);

    expect(await screen.findByRole("alert")).toHaveTextContent("Snapshot failed");
    expect(screen.getByText("Azure CLI状態取得失敗")).toBeInTheDocument();
    expect(screen.queryByText("Azure CLI未検出")).not.toBeInTheDocument();
    expect(screen.queryByText("Azure CLIをインストールしてからBridgeを再起動してください。")).not.toBeInTheDocument();
  });

  it("shows an empty state when the subscription has no Azure Model Resource", async () => {
    const app = installBoundApp("");
    app.Subscriptions.mockResolvedValue([{ id: "subscription-1", name: "Subscription 1", tenantId: "tenant-b", state: "Enabled" }]);
    render(<App />);

    await waitFor(() => expect(app.ResourceGroups).toHaveBeenCalledWith("tenant-b", "subscription-1"));
    expect(await screen.findByText("このSubscriptionにはAzure Model Resourceがありません。")).toBeInTheDocument();
  });

  it("explains Windows approval before Sync", async () => {
    await renderAndWaitForInitialLoad("");
    fireEvent.click(screen.getByRole("button", { name: "Sync" }));

    expect(screen.getByText("opencodex serviceが未登録の場合、初回のSyncでWindowsの管理者承認が表示されます。")).toBeInTheDocument();
  });
});

describe("App multi-deployment selection", () => {
  afterEach(() => {
    cleanup();
    window.go = undefined;
  });

  async function openDeployments(savedSelection: Partial<Snapshot["selection"]> = {}, ready = false) {
    const app = installDeploymentApp(savedSelection, ready);
    render(<App />);
    await waitFor(() => expect(app.Deployments).toHaveBeenCalledWith("tenant-b", "subscription-1", "group-1", "model-resource"));
    fireEvent.click(screen.getByRole("button", { name: "Deployments" }));
    return app;
  }

  it("selects the first deployment and default on first load", async () => {
    await openDeployments();

    expect(screen.getByRole("checkbox", { name: "deployment-aを公開対象にする" })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: "deployment-bを公開対象にする" })).not.toBeChecked();
    expect(screen.getByRole("radio", { name: "deployment-aを既定にする" })).toBeChecked();
  });

  it("restores saved deployment targets and default", async () => {
    await openDeployments({ resourceName: "model-resource", deploymentNames: ["deployment-b"], defaultDeploymentName: "deployment-b" });

    expect(screen.getByRole("checkbox", { name: "deployment-aを公開対象にする" })).not.toBeChecked();
    expect(screen.getByRole("checkbox", { name: "deployment-bを公開対象にする" })).toBeChecked();
    expect(screen.getByRole("radio", { name: "deployment-bを既定にする" })).toBeChecked();
  });

  it("clears the saved selection when the resource changes and selects the new first deployment", async () => {
    const app = installDeploymentApp({ resourceName: "model-resource", deploymentNames: ["deployment-b"], defaultDeploymentName: "deployment-b" });
    app.ModelResources.mockResolvedValue([
      { id: "resource-1", name: "model-resource", kind: "OpenAI", location: "japaneast", endpoint: "https://example.openai.azure.com", disableLocalAuth: false },
      { id: "resource-2", name: "other-resource", kind: "OpenAI", location: "japaneast", endpoint: "https://other.openai.azure.com", disableLocalAuth: false },
    ]);
    app.Deployments.mockImplementation((_tenant, _subscription, _group, resource) => Promise.resolve(resource === "other-resource" ? [{ ...deploymentList[0], id: "deployment-c", name: "deployment-c" }] : deploymentList));
    render(<App />);
    await waitFor(() => expect(app.Deployments).toHaveBeenCalledWith("tenant-b", "subscription-1", "group-1", "model-resource"));
    fireEvent.click(screen.getByRole("button", { name: "Deployments" }));
    fireEvent.change(screen.getByRole("combobox", { name: "Azure Model Resource" }), { target: { value: "other-resource" } });

    await waitFor(() => expect(app.Deployments).toHaveBeenCalledWith("tenant-b", "subscription-1", "group-1", "other-resource"));
    expect(screen.getByRole("checkbox", { name: "deployment-cを公開対象にする" })).toBeChecked();
    expect(screen.getByRole("radio", { name: "deployment-cを既定にする" })).toBeChecked();
  });

  it("sends all selected deployments and renders per-deployment connections", async () => {
    const app = await openDeployments({}, true);
    app.PreviewSync.mockResolvedValue({ ok: true, providerId: "az-model-resource", changes: [], message: "" });
    app.Sync.mockResolvedValue({
      ok: true,
      providerId: "az-model-resource",
      deployment: "deployment-a",
      stages: [
        { name: "catalog", status: "failed", message: "Catalog failed" },
        { name: "connection", status: "pending", message: "Test Responses endpoint" },
      ],
      connections: [
        { deployment: "deployment-a", status: "succeeded", message: "Responses endpoint is ready" },
        { deployment: "deployment-b", status: "failed", message: "deployment-b failed" },
      ],
    });
    fireEvent.click(screen.getByRole("checkbox", { name: "deployment-bを公開対象にする" }));
    fireEvent.click(screen.getByRole("button", { name: "Sync" }));
    fireEvent.click(screen.getByRole("button", { name: "差分を確認" }));
    await screen.findByText("差分確認済み");
    fireEvent.click(screen.getByRole("checkbox", { name: /実際のResponsesリクエスト/ }));
    fireEvent.click(screen.getByRole("button", { name: "Syncを明示実行" }));

    await waitFor(() => expect(app.Sync).toHaveBeenCalledWith(expect.objectContaining({
      deploymentNames: ["deployment-a", "deployment-b"],
      defaultDeploymentName: "deployment-a",
      providerId: "",
      confirmCosts: true,
    })));
    expect(await screen.findByText("Deployment別接続テスト")).toBeInTheDocument();
    expect(screen.getByText("deployment-b failed")).toBeInTheDocument();
    expect(document.querySelector(".stage .pending")).toHaveTextContent("·");
  });

  it("keeps the Provider ID when deployments change within the same resource", async () => {
    const app = await openDeployments({ resourceName: "model-resource", providerId: "az-model-resource" }, true);

    fireEvent.click(screen.getByRole("button", { name: "Sync" }));
    await waitFor(() => expect(screen.getByRole("textbox", { name: "Provider ID" })).toHaveValue("az-model-resource"));
    fireEvent.click(screen.getByRole("button", { name: "Deployments" }));
    fireEvent.click(screen.getByRole("checkbox", { name: "deployment-bを公開対象にする" }));
    fireEvent.click(screen.getByRole("button", { name: "Sync" }));

    expect(screen.getByRole("textbox", { name: "Provider ID" })).toHaveValue("az-model-resource");
  });

  it("locks the Provider ID for a resource already managed by Bridge", async () => {
    const app = installDeploymentApp({ resourceName: "model-resource", providerId: "az-model-resource" }, true);
    app.ManagedProviders.mockResolvedValue([{
      resourceId: "resource-1",
      providerId: "az-model-resource",
      tenantId: "tenant-b",
      subscriptionId: "subscription-1",
      resourceGroup: "group-1",
      resourceName: "model-resource",
      location: "japaneast",
      deploymentNames: ["deployment-a"],
      defaultDeploymentName: "deployment-a",
    }]);
    render(<App />);

    await waitFor(() => expect(app.ManagedProviders).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole("button", { name: "Sync" }));

    const providerInput = screen.getByRole("textbox", { name: "Provider ID" });
    expect(providerInput).toHaveValue("az-model-resource");
    expect(providerInput).toBeDisabled();
    expect(screen.getByText(/Provider IDはDisconnectまで変更できません。/)).toBeInTheDocument();
  });

  it("requires a current successful preview before Sync", async () => {
    const app = await openDeployments({}, true);
    app.PreviewSync.mockResolvedValue({
      ok: true,
      providerId: "az-model-resource",
      changes: [
        { area: "credential", action: "refresh", details: "Register current key" },
        { area: "connection", action: "test", details: "Send Responses request" },
      ],
      message: "",
    });
    app.Sync.mockResolvedValue({ ok: true, providerId: "az-model-resource", deployment: "deployment-a", stages: [], connections: [] });
    fireEvent.click(screen.getByRole("button", { name: "Sync" }));

    const syncButton = screen.getByRole("button", { name: "Syncを明示実行" });
    expect(syncButton).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "差分を確認" }));
    await screen.findByText("予定操作");
    expect(syncButton).toBeDisabled();

    fireEvent.click(screen.getByRole("checkbox", { name: /実際のResponsesリクエスト/ }));
    expect(syncButton).toBeEnabled();
    expect(screen.getByText(/Register current key/)).toBeInTheDocument();
  });

  it("keeps the successful Disconnect result visible", async () => {
    const app = installDeploymentApp({}, true);
    const managed = {
      resourceId: "resource-1", providerId: "az-model-resource", tenantId: "tenant-b",
      subscriptionId: "subscription-1", resourceGroup: "group-1", resourceName: "model-resource",
      location: "japaneast", deploymentNames: ["deployment-a"], defaultDeploymentName: "deployment-a",
    };
    app.ManagedProviders.mockResolvedValue([managed]);
    app.PreviewDisconnect.mockResolvedValue({ ok: true, managed, isDefault: false, dependentCombos: [], replacementProviders: ["openai"], message: "" });
    app.Disconnect.mockResolvedValue({ ok: true, stages: [{ name: "provider", status: "succeeded", message: "Removed" }] });
    render(<App />);
    await waitFor(() => expect(app.ManagedProviders).toHaveBeenCalled());
    fireEvent.click(screen.getByRole("button", { name: "Sync" }));
    fireEvent.click(screen.getByRole("button", { name: "Disconnectを確認" }));
    await waitFor(() => expect(app.PreviewDisconnect).toHaveBeenCalledWith("resource-1"));
    fireEvent.click(screen.getByRole("checkbox", { name: /Azure資源を削除せずにDisconnect/ }));
    fireEvent.click(screen.getByRole("button", { name: "Disconnectを実行" }));

    expect(await screen.findByRole("heading", { name: "操作完了" })).toBeInTheDocument();
    expect(screen.getByText("Removed")).toBeInTheDocument();
  });

  it("requires explicit confirmation before creating a deployment and shows Sync required after success", async () => {
    const app = await openDeployments();
    app.CreateDeployment.mockResolvedValue({
      ok: true,
      operation: "create",
      status: "succeeded",
      deployment: { ...deploymentList[0], name: "codex-deployment" },
      message: "Azure deployment operation completed",
      syncRequired: true,
    });

    fireEvent.change(screen.getByRole("textbox", { name: "Deployment name" }), { target: { value: "codex-deployment" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Model" }), { target: { value: "gpt-4o" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Model version" }), { target: { value: "2024-08-06" } });
    fireEvent.change(screen.getByRole("combobox", { name: "SKU" }), { target: { value: "Standard" } });
    expect(screen.getByText(/最小 1000 \/ 最大 150000 \/ 刻み 1000 \/ 既定 10000/)).toBeInTheDocument();
    fireEvent.change(screen.getByRole("combobox", { name: "Capacity (TPM)" }), { target: { value: "150000" } });

    const createButton = screen.getByRole("button", { name: "Deploymentを作成" });
    expect(createButton).toBeDisabled();
    fireEvent.click(screen.getByRole("checkbox", { name: /Azureで作成・更新/ }));
    expect(createButton).toBeEnabled();
    fireEvent.click(createButton);

    await waitFor(() => expect(app.CreateDeployment).toHaveBeenCalledWith(expect.objectContaining({
      deploymentName: "codex-deployment",
      modelName: "gpt-4o",
      modelFormat: "OpenAI",
      modelVersion: "2024-08-06",
      sku: "Standard",
      capacity: 150,
      confirm: true,
    })));
    expect(app.CreateDeployment.mock.calls[0][0]).not.toHaveProperty("versionUpgradeOption");
    expect(await screen.findByRole("status", { name: "Sync required" })).toHaveTextContent("明示Syncが必要");
    expect(app.Sync).not.toHaveBeenCalled();
  });

  it("keeps Provisioned capacity in PTU without TPM conversion", async () => {
    const app = await openDeployments();
    app.CreateDeployment.mockResolvedValue({
      ok: true,
      operation: "create",
      status: "succeeded",
      deployment: { ...deploymentList[0], name: "ptu-deployment", sku: "ProvisionedManaged", capacity: 8 },
      message: "Azure deployment operation completed",
      syncRequired: true,
    });

    fireEvent.change(screen.getByRole("textbox", { name: "Deployment name" }), { target: { value: "ptu-deployment" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Model" }), { target: { value: "gpt-4o-ptu" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Model version" }), { target: { value: "2024-08-06" } });
    fireEvent.change(screen.getByRole("combobox", { name: "SKU" }), { target: { value: "ProvisionedManaged" } });
    expect(screen.getByText(/Capacity単位: PTU/)).toBeInTheDocument();
    const capacity = screen.getByRole("spinbutton", { name: "Capacity (PTU)" });
    expect(capacity).toHaveValue(4);
    fireEvent.change(capacity, { target: { value: "8" } });
    fireEvent.click(screen.getByRole("checkbox", { name: /Azureで作成・更新/ }));
    fireEvent.click(screen.getByRole("button", { name: "Deploymentを作成" }));

    await waitFor(() => expect(app.CreateDeployment).toHaveBeenCalledWith(expect.objectContaining({
      sku: "ProvisionedManaged",
      capacity: 8,
      confirm: true,
    })));
  });

  it("does not submit a Standard capacity that is not divisible by its TPM multiplier", async () => {
    const app = await openDeployments();

    fireEvent.change(screen.getByRole("textbox", { name: "Deployment name" }), { target: { value: "invalid-capacity" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Model" }), { target: { value: "gpt-4.1" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Model version" }), { target: { value: "2025-04-14" } });
    fireEvent.change(screen.getByRole("combobox", { name: "SKU" }), { target: { value: "Standard" } });
    const capacity = screen.getByRole("spinbutton", { name: "Capacity (TPM)" });
    fireEvent.change(capacity, { target: { value: "10001" } });
    fireEvent.click(screen.getByRole("checkbox", { name: /Azureで作成・更新/ }));

    expect(screen.getByRole("button", { name: "Deploymentを作成" })).toBeDisabled();
    expect(app.CreateDeployment).not.toHaveBeenCalled();
  });

  it("shows the current and changed values before updating a deployment", async () => {
    const app = await openDeployments();
    app.UpdateDeployment.mockResolvedValue({
      ok: true,
      operation: "update",
      status: "succeeded",
      deployment: deploymentList[0],
      message: "Azure deployment operation completed",
      syncRequired: true,
    });

    fireEvent.change(screen.getByRole("combobox", { name: "編集対象Deployment" }), { target: { value: "deployment-a" } });
    const comparison = screen.getByRole("heading", { name: "更新前後の比較" }).parentElement as HTMLElement;
    expect(comparison).toBeInTheDocument();
    expect(within(comparison).getAllByText("NoAutoUpgrade")).toHaveLength(2);
    expect(within(comparison).getAllByText("10000 TPM")).toHaveLength(2);
    fireEvent.change(screen.getByRole("combobox", { name: "Capacity (TPM)" }), { target: { value: "100000" } });
    fireEvent.click(screen.getByRole("checkbox", { name: /Azureで作成・更新/ }));
    fireEvent.click(screen.getByRole("button", { name: "Deploymentを更新" }));

    await waitFor(() => expect(app.UpdateDeployment).toHaveBeenCalledWith(expect.objectContaining({
      deploymentName: "deployment-a",
      capacity: 100,
      versionUpgradeOption: "NoAutoUpgrade",
      confirm: true,
    })));
  });

  it("shows the existing and changed capacities with their own units when SKU changes", async () => {
    const app = await openDeployments();
    fireEvent.change(screen.getByRole("combobox", { name: "編集対象Deployment" }), { target: { value: "deployment-a" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Model" }), { target: { value: "gpt-4o-ptu" } });
    fireEvent.change(screen.getByRole("combobox", { name: "Model version" }), { target: { value: "2024-08-06" } });
    fireEvent.change(screen.getByRole("combobox", { name: "SKU" }), { target: { value: "ProvisionedManaged" } });

    const comparison = screen.getByRole("heading", { name: "更新前後の比較" }).parentElement as HTMLElement;
    expect(within(comparison).getByText("10000 TPM")).toBeInTheDocument();
    expect(within(comparison).getByText("4 PTU")).toBeInTheDocument();
    expect(within(comparison).getByText("Capacity")).toBeInTheDocument();
    expect(app.UpdateDeployment).not.toHaveBeenCalled();
  });
});
