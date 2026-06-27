import { describe, expect, it } from "vitest";
import {
  formatDocPhaseLogLine,
  formatDocPhaseRunningLine,
  sectionTransitionLogLine,
} from "./docPhase";

describe("formatDocPhaseRunningLine", () => {
  it("formats section_enhance with section", () => {
    const line = formatDocPhaseRunningLine(
      {
        doc_key: "architecture",
        step: "section_enhance",
        section: "4. Layers",
      },
      "Architecture",
    );
    expect(line).toBe("Architecture · §4. Layers — enhancing…");
  });

  it("formats coherence_audit with findings", () => {
    const line = formatDocPhaseRunningLine(
      {
        doc_key: "architecture",
        step: "coherence_audit",
        findings_count: 2,
      },
      "Architecture",
    );
    expect(line).toBe("Architecture — 2 consistency issue(s) found");
  });

  it("formats coherence_fix", () => {
    const line = formatDocPhaseRunningLine(
      {
        doc_key: "plan",
        step: "coherence_fix",
        section: "5. Implementation Tasks",
      },
      "Plan",
    );
    expect(line).toBe("Plan · §5. Implementation Tasks — fixing consistency…");
  });

  it("falls back to detail for draft step", () => {
    const line = formatDocPhaseRunningLine(
      {
        doc_key: "readme",
        step: "draft",
        detail: "Generating first draft with AI…",
      },
      "README",
    );
    expect(line).toContain("README");
    expect(line).toContain("Generating first draft");
  });
});

describe("sectionTransitionLogLine", () => {
  it("logs done when section changes", () => {
    const line = sectionTransitionLogLine(
      { docKey: "architecture", section: "3. Scope" },
      {
        doc_key: "architecture",
        step: "section_enhance",
        section: "4. Layers",
      },
      "Architecture",
    );
    expect(line).toBe("Architecture · §3. Scope — done");
  });
});

describe("formatDocPhaseLogLine", () => {
  it("formats complete step", () => {
    const line = formatDocPhaseLogLine(
      {
        doc_key: "architecture",
        step: "complete",
        detail: "Document generated successfully.",
      },
      "Architecture",
    );
    expect(line).toBe("✦ Architecture — Document generated successfully.");
  });
});
