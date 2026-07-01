import { describe, expect, it } from "vitest";
import { isServerPassComplete } from "./sessionPassSync";
import type { Session } from "$lib/types";

function sess(
  status: Session["status"],
  iteration: number,
): Session {
  return {
    id: "s1",
    idea: "test",
    status,
    max_iterations: 5,
    output_docs: [],
    current_state: {
      idea: {},
      architecture: {},
      execution_plan: [],
      risks: [],
      assumptions: [],
      open_questions: [],
      metrics: { confidence: 0.9 },
      meta: { iteration, agents: [] },
    },
    created_at: "",
    updated_at: "",
  };
}

describe("isServerPassComplete", () => {
  it("returns false while status is running", () => {
    expect(isServerPassComplete(sess("running", 1), 2)).toBe(false);
  });

  it("returns true when active and iteration reached expected pass", () => {
    expect(isServerPassComplete(sess("active", 2), 2)).toBe(true);
  });

  it("returns false when active but iteration not advanced yet", () => {
    expect(isServerPassComplete(sess("active", 1), 2)).toBe(false);
  });

  it("returns true for converged sessions", () => {
    expect(isServerPassComplete(sess("converged", 5), 5)).toBe(true);
  });
});
