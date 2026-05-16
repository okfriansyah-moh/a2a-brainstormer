/**
 * API service layer — all HTTP calls to the backend REST API.
 *
 * Security rules:
 *  - No secrets or credentials are ever included in requests (the backend
 *    resolves credentials from env vars server-side).
 *  - The base URL is configurable via the VITE_API_BASE_URL env var so that
 *    dev, staging, and production can point to different backends without
 *    rebuilding the frontend.
 *  - Throws `ApiError` on any non-2xx response so callers can handle errors
 *    uniformly without inspecting raw Response objects.
 *
 * Endpoint reference: docs/PLAN.md §8.7
 */

import type {
  Agent,
  CreateAgentRequest,
  CreateSessionRequest,
  CreateSessionResponse,
  CreateSkillRequest,
  IterateResponse,
  Session,
  Skill,
  UpdateAgentRequest,
  UpdateSkillRequest,
} from "$lib/types";

// ── Configuration ─────────────────────────────────────────────────────────────

/**
 * Backend base URL. Set VITE_API_BASE_URL in .env to override.
 * Never embed credentials or API keys in this URL.
 */
const API_BASE: string =
  import.meta.env.VITE_API_BASE_URL || "http://localhost:8080";

// ── Error type ────────────────────────────────────────────────────────────────

/** Structured error thrown on non-2xx responses. */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly body: string,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// ── Internal helpers ──────────────────────────────────────────────────────────

/**
 * Execute a fetch request and return the parsed JSON body.
 * Throws ApiError on non-2xx status codes.
 * Prevents SSRF: the base URL is read from a build-time env var, not from
 * user input. Path segments come only from this module's own functions.
 */
async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const url = `${API_BASE}${path}`;
  const response = await fetch(url, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });

  if (!response.ok) {
    const body = await response.text();
    throw new ApiError(
      response.status,
      body,
      `API request failed: ${response.status} ${response.statusText}`,
    );
  }

  // 204 No Content — return empty object cast to T
  if (response.status === 204) {
    return {} as T;
  }

  return response.json() as Promise<T>;
}

function json(body: unknown): RequestInit {
  return { body: JSON.stringify(body) };
}

// ── Sessions (§8.7) ───────────────────────────────────────────────────────────

/**
 * Create a new brainstorm session.
 * Requires at least 2 agent_ids — the backend enforces this with HTTP 400.
 */
export async function createSession(
  req: CreateSessionRequest,
): Promise<CreateSessionResponse> {
  return request<CreateSessionResponse>("/sessions", {
    method: "POST",
    ...json(req),
  });
}

/** Get a session by ID, including its current canonical state. */
export async function getSession(sessionId: string): Promise<Session> {
  return request<Session>(`/sessions/${encodeURIComponent(sessionId)}`);
}

/**
 * Trigger one iteration of the N-agent pipeline for the given session.
 * Returns the updated canonical state after the full pipeline pass.
 */
export async function iterate(sessionId: string): Promise<IterateResponse> {
  return request<IterateResponse>(
    `/sessions/${encodeURIComponent(sessionId)}/iterate`,
    {
      method: "POST",
    },
  );
}

/**
 * Finalize (approve) a session — triggers markdown artifact generation
 * and transitions the session to `approved` status.
 */
export async function finalizeSession(sessionId: string): Promise<Session> {
  return request<Session>(
    `/sessions/${encodeURIComponent(sessionId)}/finalize`,
    {
      method: "POST",
    },
  );
}

// ── Agents (§8.7) ─────────────────────────────────────────────────────────────

/** List all registered agents (each includes their skills[]). */
export async function getAgents(): Promise<Agent[]> {
  return request<Agent[]>("/agents");
}

/** Get a single agent by ID (includes skills[]). */
export async function getAgent(agentId: string): Promise<Agent> {
  return request<Agent>(`/agents/${encodeURIComponent(agentId)}`);
}

/** Register a new agent. */
export async function createAgent(req: CreateAgentRequest): Promise<Agent> {
  return request<Agent>("/agents", { method: "POST", ...json(req) });
}

/** Update an existing agent's fields. */
export async function updateAgent(
  agentId: string,
  req: UpdateAgentRequest,
): Promise<Agent> {
  return request<Agent>(`/agents/${encodeURIComponent(agentId)}`, {
    method: "PUT",
    ...json(req),
  });
}

/** Delete an agent by ID. */
export async function deleteAgent(agentId: string): Promise<void> {
  return request<void>(`/agents/${encodeURIComponent(agentId)}`, {
    method: "DELETE",
  });
}

// ── Skills (§8.7) ─────────────────────────────────────────────────────────────

/** List all skills in the library. */
export async function getSkills(): Promise<Skill[]> {
  return request<Skill[]>("/skills");
}

/** Get a single skill by ID. */
export async function getSkill(skillId: string): Promise<Skill> {
  return request<Skill>(`/skills/${encodeURIComponent(skillId)}`);
}

/** Create a new skill. */
export async function createSkill(req: CreateSkillRequest): Promise<Skill> {
  return request<Skill>("/skills", { method: "POST", ...json(req) });
}

/** Update an existing skill. */
export async function updateSkill(
  skillId: string,
  req: UpdateSkillRequest,
): Promise<Skill> {
  return request<Skill>(`/skills/${encodeURIComponent(skillId)}`, {
    method: "PUT",
    ...json(req),
  });
}

/** Delete a skill by ID. */
export async function deleteSkill(skillId: string): Promise<void> {
  return request<void>(`/skills/${encodeURIComponent(skillId)}`, {
    method: "DELETE",
  });
}

// ── Agent–Skill attachments (§8.7) ────────────────────────────────────────────

/** Get all skills currently attached to an agent. */
export async function getAgentSkills(agentId: string): Promise<Skill[]> {
  return request<Skill[]>(`/agents/${encodeURIComponent(agentId)}/skills`);
}

/** Attach a skill to an agent (idempotent). */
export async function attachSkill(
  agentId: string,
  skillId: string,
): Promise<void> {
  return request<void>(
    `/agents/${encodeURIComponent(agentId)}/skills/${encodeURIComponent(skillId)}`,
    { method: "POST" },
  );
}

/** Detach a skill from an agent. */
export async function detachSkill(
  agentId: string,
  skillId: string,
): Promise<void> {
  return request<void>(
    `/agents/${encodeURIComponent(agentId)}/skills/${encodeURIComponent(skillId)}`,
    { method: "DELETE" },
  );
}
