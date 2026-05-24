/**
 * Unit tests for frontend/src/lib/services/api.ts
 *
 * Strategy: mock the global `fetch` to simulate backend responses.
 * All tests run without network access (§8.15: "Tests without network").
 *
 * Coverage (Task 15 requirements):
 *  - 400 Bad Request → ApiError with status 400
 *  - 404 Not Found   → ApiError with status 404
 *  - 500 Server Error → ApiError with status 500
 *  - 2xx success     → parsed response returned
 *  - 204 No Content  → empty object returned (delete endpoints)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  ApiError,
  createSession,
  getSession,
  iterate,
  finalizeSession,
  listSessions,
  getAgents,
  createAgent,
  updateAgent,
  deleteAgent,
  getSkills,
  createSkill,
  updateSkill,
  deleteSkill,
  attachSkill,
  detachSkill,
  getAgentSkills,
} from "./api";

// ── fetch mock helpers ────────────────────────────────────────────────────────

function mockFetch(status: number, body: unknown): void {
  const bodyText = typeof body === "string" ? body : JSON.stringify(body);
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({
      ok: status >= 200 && status < 300,
      status,
      statusText: String(status),
      text: () => Promise.resolve(bodyText),
      json: () => Promise.resolve(body),
    }),
  );
}

beforeEach(() => {
  vi.restoreAllMocks();
});

// ── ApiError shape ────────────────────────────────────────────────────────────

describe("ApiError", () => {
  it("has name, status, body, message", () => {
    const err = new ApiError(404, "not found body", "API request failed: 404");
    expect(err.name).toBe("ApiError");
    expect(err.status).toBe(404);
    expect(err.body).toBe("not found body");
    expect(err.message).toContain("404");
  });
});

// ── Error response handling ───────────────────────────────────────────────────

describe("error response handling", () => {
  it("throws ApiError on 400", async () => {
    mockFetch(400, "validation error");
    await expect(
      createSession({ idea: "", agent_ids: [], max_iterations: 1 }),
    ).rejects.toMatchObject({ status: 400, name: "ApiError" });
  });

  it("throws ApiError on 404", async () => {
    mockFetch(404, "not found");
    await expect(getSession("nonexistent-id")).rejects.toMatchObject({
      status: 404,
      name: "ApiError",
    });
  });

  it("throws ApiError on 500", async () => {
    mockFetch(500, "internal server error");
    await expect(getAgents()).rejects.toMatchObject({
      status: 500,
      name: "ApiError",
    });
  });

  it("throws ApiError on 409 conflict", async () => {
    mockFetch(409, "conflict");
    await expect(
      createAgent({
        name: "dup",
        description: "A duplicate agent.",
        default_role: "build",
        system_prompt: "p",
        endpoint: "http://a",
        llm_config: {
          provider: "copilot",
          model: "gpt-4o",
          credential_ref: "K",
        },
      }),
    ).rejects.toMatchObject({ status: 409, name: "ApiError" });
  });
});

// ── createSession ─────────────────────────────────────────────────────────────

describe("createSession", () => {
  it("returns CreateSessionResponse on 201", async () => {
    const payload = { id: "sess-1", idea: "test", status: "active" };
    mockFetch(201, payload);
    const result = await createSession({
      idea: "test",
      agent_ids: ["a1", "a2"],
      max_iterations: 5,
    });
    expect(result.id).toBe("sess-1");
  });
});

// ── getSession ────────────────────────────────────────────────────────────────

describe("getSession", () => {
  it("returns Session on 200", async () => {
    const session = { id: "s1", idea: "idea", status: "active" };
    mockFetch(200, session);
    const result = await getSession("s1");
    expect(result.id).toBe("s1");
  });
});

// ── iterate ───────────────────────────────────────────────────────────────────

describe("iterate", () => {
  it("returns IterateResponse on 200", async () => {
    const payload = { state: { idea: {} }, converged: false, iteration: 1 };
    mockFetch(200, payload);
    const result = await iterate("sess-1");
    expect(result.converged).toBe(false);
    expect(result.iteration).toBe(1);
  });

  it("throws on 400 (session not found)", async () => {
    mockFetch(400, "bad session");
    await expect(iterate("bad-id")).rejects.toMatchObject({ status: 400 });
  });
});

// ── finalizeSession ───────────────────────────────────────────────────────────

describe("finalizeSession", () => {
  it("resolves on 204", async () => {
    mockFetch(204, "");
    await expect(finalizeSession("s1")).resolves.toBeDefined();
  });

  it("returns FinalizeResponse with documents on 200", async () => {
    const response = {
      session_id: "s1",
      documents: {
        architecture: {
          filename: "architecture.md",
          content: "# Architecture\n\nDetails here.",
          line_count: 3,
        },
        roadmap: {
          filename: "roadmap.md",
          content: "# Roadmap\n\nPhase 1: ...",
          line_count: 3,
        },
      },
      status: "approved",
    };
    mockFetch(200, response);
    const result = await finalizeSession("s1");
    expect(result.session_id).toBe("s1");
    expect(result.documents["architecture"].content).toContain("# Architecture");
    expect(result.documents["roadmap"].content).toContain("# Roadmap");
    expect(result.status).toBe("approved");
  });

  it("throws ApiError on 404 (session not found)", async () => {
    mockFetch(404, "session not found");
    await expect(finalizeSession("missing-id")).rejects.toMatchObject({
      status: 404,
      name: "ApiError",
    });
  });
});

// ── listSessions ──────────────────────────────────────────────────────────────

describe("listSessions", () => {
  it("returns empty sessions array when backend returns empty list", async () => {
    const response = { sessions: [], total: 0 };
    mockFetch(200, response);
    const result = await listSessions();
    expect(result.sessions).toEqual([]);
    expect(result.total).toBe(0);
  });

  it("returns populated sessions array with all fields", async () => {
    const response = {
      sessions: [
        {
          id: "s1",
          idea: "Build a SaaS pricing model",
          status: "converged",
          max_iterations: 7,
          current_iteration: 5,
          confidence: 0.87,
          agent_count: 3,
          created_at: "2024-01-15T10:00:00Z",
          updated_at: "2024-01-15T11:30:00Z",
        },
        {
          id: "s2",
          idea: "Design a microservice architecture",
          status: "active",
          max_iterations: 5,
          current_iteration: 2,
          confidence: 0.42,
          agent_count: 2,
          created_at: "2024-01-16T09:00:00Z",
          updated_at: "2024-01-16T09:45:00Z",
        },
      ],
      total: 2,
    };
    mockFetch(200, response);
    const result = await listSessions();
    expect(result.sessions).toHaveLength(2);
    expect(result.total).toBe(2);
    expect(result.sessions[0].id).toBe("s1");
    expect(result.sessions[0].confidence).toBe(0.87);
    expect(result.sessions[1].status).toBe("active");
  });

  it("throws ApiError on 500", async () => {
    mockFetch(500, "internal error");
    await expect(listSessions()).rejects.toMatchObject({
      status: 500,
      name: "ApiError",
    });
  });
});

// ── getAgents ─────────────────────────────────────────────────────────────────

describe("getAgents", () => {
  it("returns agent array on 200", async () => {
    const agents = [
      { id: "a1", name: "Alpha" },
      { id: "a2", name: "Beta" },
    ];
    mockFetch(200, agents);
    const result = await getAgents();
    expect(result).toHaveLength(2);
    expect(result[0].id).toBe("a1");
  });

  it("returns empty array when backend returns []", async () => {
    mockFetch(200, []);
    const result = await getAgents();
    expect(result).toEqual([]);
  });
});

// ── createAgent ───────────────────────────────────────────────────────────────

describe("createAgent", () => {
  it("returns created Agent on 201", async () => {
    const agent = { id: "a1", name: "New Agent", default_role: "build" };
    mockFetch(201, agent);
    const result = await createAgent({
      name: "New Agent",
      description: "A builder agent.",
      default_role: "build",
      system_prompt: "You are a builder.",
      endpoint: "http://localhost:9090",
      llm_config: {
        provider: "copilot",
        model: "gpt-4o",
        credential_ref: "COPILOT_API_KEY",
      },
    });
    expect(result.id).toBe("a1");
  });
});

// ── updateAgent ───────────────────────────────────────────────────────────────

describe("updateAgent", () => {
  it("returns updated Agent on 200", async () => {
    const agent = { id: "a1", name: "Updated Agent" };
    mockFetch(200, agent);
    const result = await updateAgent("a1", { name: "Updated Agent" });
    expect(result.name).toBe("Updated Agent");
  });
});

// ── deleteAgent ───────────────────────────────────────────────────────────────

describe("deleteAgent", () => {
  it("resolves on 204", async () => {
    mockFetch(204, "");
    await expect(deleteAgent("a1")).resolves.toBeDefined();
  });

  it("throws ApiError on 404", async () => {
    mockFetch(404, "not found");
    await expect(deleteAgent("missing")).rejects.toMatchObject({ status: 404 });
  });
});

// ── getSkills ─────────────────────────────────────────────────────────────────

describe("getSkills", () => {
  it("returns skill array on 200", async () => {
    const skills = [{ id: "sk1", name: "Security", prompt: "OWASP" }];
    mockFetch(200, skills);
    const result = await getSkills();
    expect(result[0].id).toBe("sk1");
  });
});

// ── createSkill ───────────────────────────────────────────────────────────────

describe("createSkill", () => {
  it("returns Skill on 201", async () => {
    const skill = { id: "sk1", name: "Security Review", prompt: "..." };
    mockFetch(201, skill);
    const result = await createSkill({
      name: "Security Review",
      description: "OWASP security review.",
      prompt: "...",
    });
    expect(result.id).toBe("sk1");
  });
});

// ── updateSkill ───────────────────────────────────────────────────────────────

describe("updateSkill", () => {
  it("returns updated Skill on 200", async () => {
    const skill = { id: "sk1", name: "Updated Skill", prompt: "new prompt" };
    mockFetch(200, skill);
    const result = await updateSkill("sk1", {
      name: "Updated Skill",
      prompt: "new prompt",
    });
    expect(result.name).toBe("Updated Skill");
  });
});

// ── deleteSkill ───────────────────────────────────────────────────────────────

describe("deleteSkill", () => {
  it("resolves on 204", async () => {
    mockFetch(204, "");
    await expect(deleteSkill("sk1")).resolves.toBeDefined();
  });
});

// ── attachSkill / detachSkill ─────────────────────────────────────────────────

describe("attachSkill", () => {
  it("resolves on 204", async () => {
    mockFetch(204, "");
    await expect(attachSkill("a1", "sk1")).resolves.toBeDefined();
  });

  it("throws 404 when agent or skill not found", async () => {
    mockFetch(404, "not found");
    await expect(attachSkill("bad-agent", "sk1")).rejects.toMatchObject({
      status: 404,
    });
  });
});

describe("detachSkill", () => {
  it("resolves on 204", async () => {
    mockFetch(204, "");
    await expect(detachSkill("a1", "sk1")).resolves.toBeDefined();
  });
});

// ── getAgentSkills ────────────────────────────────────────────────────────────

describe("getAgentSkills", () => {
  it("returns skill array on 200", async () => {
    const skills = [{ id: "sk1", name: "skill", prompt: "p" }];
    mockFetch(200, skills);
    const result = await getAgentSkills("a1");
    expect(result).toHaveLength(1);
  });
});
