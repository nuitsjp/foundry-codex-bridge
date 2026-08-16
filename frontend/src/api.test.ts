import { describe, expect, it, vi } from "vitest";
import { api, normalizeList } from "./api";

describe("list API normalization", () => {
  it("converts null and undefined to empty arrays", () => {
    expect(normalizeList(null)).toEqual([]);
    expect(normalizeList(undefined)).toEqual([]);
  });

  it("normalizes every Wails list method", async () => {
    const app = {
      Tenants: vi.fn().mockResolvedValue(null),
      Subscriptions: vi.fn().mockResolvedValue(undefined),
      ResourceGroups: vi.fn().mockResolvedValue(null),
      ModelResources: vi.fn().mockResolvedValue(undefined),
      Deployments: vi.fn().mockResolvedValue(null),
      Models: vi.fn().mockResolvedValue(undefined),
    };
    window.go = { main: { App: app as never } };

    await expect(api.tenants()).resolves.toEqual([]);
    await expect(api.subscriptions("tenant")).resolves.toEqual([]);
    await expect(api.resourceGroups("tenant", "subscription")).resolves.toEqual([]);
    await expect(api.resources("tenant", "subscription", "group")).resolves.toEqual([]);
    await expect(api.deployments("tenant", "subscription", "group", "resource")).resolves.toEqual([]);
    await expect(api.models("tenant", "subscription", "group", "resource")).resolves.toEqual([]);
  });
});
