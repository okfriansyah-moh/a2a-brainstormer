import { describe, expect, it } from "vitest";
import { presentStreamBuffer, waitingPresentation } from "./streamPresenter";

describe("streamPresenter", () => {
  it("returns waiting copy when buffer is empty", () => {
    const p = waitingPresentation("Mapping layers…", "build");
    expect(p.headline).toBe("Mapping layers…");
    expect(p.hasContent).toBe(false);
  });

  it("detects sections from partial JSON", () => {
    const p = presentStreamBuffer(
      '{"architecture":{"layers":[{"name":"API Gateway"',
      "",
      "build",
    );
    expect(p.hasContent).toBe(true);
    expect(p.headline).toContain("architecture");
  });

  it("extracts human snippets from string fields", () => {
    const p = presentStreamBuffer(
      '{"execution_plan":[{"title":"Phase 1: Auth","description":"OAuth2',
      "",
      "build",
    );
    expect(p.bullets.some((b) => b.includes("Phase 1"))).toBe(true);
  });
});
