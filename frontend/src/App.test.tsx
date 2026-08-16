import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
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
      deploymentName: "",
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
  };
  window.go = { main: { App: app as never } };
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
