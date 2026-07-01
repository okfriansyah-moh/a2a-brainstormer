import { describe, expect, it } from "vitest";
import { isGenericDocRunningLine } from "./docLoadingPhrases";

describe("isGenericDocRunningLine", () => {
  it("treats null as generic", () => {
    expect(isGenericDocRunningLine(null)).toBe(true);
  });

  it("detects static Generating label placeholder", () => {
    expect(isGenericDocRunningLine("Generating Architecture…")).toBe(true);
  });

  it("keeps section-specific SSE lines", () => {
    expect(
      isGenericDocRunningLine("Architecture · §3 — enhancing…"),
    ).toBe(false);
  });
});
