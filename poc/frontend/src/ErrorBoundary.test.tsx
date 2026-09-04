import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import ErrorBoundary from "./ErrorBoundary";

function ThrowingView(): ReactNode {
  throw new Error("render failed");
}

describe("ErrorBoundary", () => {
  beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => undefined);
  });
  afterEach(() => vi.restoreAllMocks());

  it("shows a reload action when a child throws during rendering", () => {
    render(
      <ErrorBoundary>
        <ThrowingView />
      </ErrorBoundary>,
    );

    expect(screen.getByRole("alert")).toHaveTextContent("画面の表示に失敗しました");
    expect(screen.getByRole("button", { name: "再読み込み" })).toBeInTheDocument();
  });
});
