import { cleanup, render, screen, waitFor } from "@testing-library/react";
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
});
