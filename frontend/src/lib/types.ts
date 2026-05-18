/**
 * TypeScript types for the a2a-brainstorm frontend.
 * Mirrors the canonical state model defined in docs/PLAN.md §8.1 and the
 * API contracts in §8.7. Do not add fields that don't exist in the backend
 * models — this file is the single source of truth for all frontend types.
 */

// ── LLM configuration ────────────────────────────────────────────────────────

/** Resolved per-dispatch LLM configuration. CredentialRef is an env var name. */
export interface LLMConfig {
  provider: string; // "copilot" | "claude"
  model: string; // e.g. "gpt-4o", "claude-opus-4"
  credential_ref: string; // env var name only — never the raw key
}

// ── Skill ────────────────────────────────────────────────────────────────────

export interface Skill {
  id: string;
  name: string;
  description: string;
  prompt: string;
  created_at: string;
}

// ── Agent ────────────────────────────────────────────────────────────────────

export interface Agent {
  id: string;
  name: string;
  description: string;
  default_role: string;
  system_prompt: string;
  llm_config: LLMConfig;
  endpoint: string;
  skills: Skill[];
  created_at: string;
}

// ── Session agent (pipeline slot binding) ────────────────────────────────────

/**
 * A session-scoped agent slot — one row in session_agents.
 * Includes the agent's output for the current iteration when present.
 */
export interface SessionAgent {
  id: string; // agent UUID
  name: string;
  role: string; // role assigned to this agent in this session
  provider: string;
  model: string;
  skills: string[]; // active skill names for this session (observability)
  output?: CanonicalState; // last output from this agent in the current iteration
}

// ── Canonical state model (§8.1) ─────────────────────────────────────────────

/** Idea captured at session creation. */
export interface Idea {
  text?: string;
  [key: string]: unknown;
}

/** Architecture section produced by the build role. */
export interface Architecture {
  overview?: string;
  components?: string[];
  decisions?: string[];
  [key: string]: unknown;
}

/** A single step in the execution plan. */
export interface ExecutionStep {
  id?: string;
  title: string;
  description: string;
  owner?: string;
  duration?: string;
  dependencies?: string[];
}

/** A risk item with severity classification. */
export interface Risk {
  id?: string;
  title: string;
  description: string;
  severity: "low" | "medium" | "high" | "critical";
  resolved?: boolean;
}

/** Confidence and quality metrics. */
export interface StateMetrics {
  confidence: number; // [0.0, 1.0]
}

/** Per-agent metadata for observability. */
export interface AgentMeta {
  agent_id: string;
  name: string;
  role: string;
  provider: string;
  model: string;
  skills: string[]; // names only
}

/** Iteration-level metadata. */
export interface StateMeta {
  iteration: number;
  agents: AgentMeta[];
}

/**
 * CanonicalState — the single shared state flowing through the iteration
 * pipeline. Matches the JSON shape in docs/PLAN.md §8.1 exactly.
 */
export interface CanonicalState {
  idea: Idea;
  architecture: Architecture;
  execution_plan: ExecutionStep[];
  risks: Risk[];
  assumptions: string[];
  open_questions: string[];
  metrics: StateMetrics;
  meta: StateMeta;
}

// ── Session ──────────────────────────────────────────────────────────────────

export interface Session {
  id: string;
  idea: string;
  status: "active" | "converged" | "approved" | "failed";
  max_iterations: number;
  current_state: CanonicalState | null;
  created_at: string;
  updated_at: string;
}

// ── API request / response shapes (§8.7) ─────────────────────────────────────

/** Body for POST /sessions */
export interface CreateSessionRequest {
  idea: string;
  agent_ids: string[]; // min 2 required
  max_iterations?: number;
  role_overrides?: Record<string, string>;
  llm_overrides?: Record<string, Partial<LLMConfig>>;
  skill_overrides?: Record<string, string[]>;
}

/** Response from POST /sessions */
export interface CreateSessionResponse {
  id: string;
  idea: string;
  status: string;
  max_iterations: number;
  created_at: string;
}

/** Response from POST /sessions/{id}/iterate */
export interface IterateResponse {
  session_id: string;
  iteration: number;
  state: CanonicalState;
  converged: boolean;
}

/** Request body for POST /agents (register agent) */
export interface CreateAgentRequest {
  name: string;
  description: string;
  default_role: string;
  system_prompt: string;
  llm_config: LLMConfig;
  endpoint: string;
}

/** Request body for PUT /agents/{id} */
export interface UpdateAgentRequest {
  name?: string;
  description?: string;
  default_role?: string;
  system_prompt?: string;
  llm_config?: Partial<LLMConfig>;
  endpoint?: string;
}

/** Request body for POST /skills */
export interface CreateSkillRequest {
  name: string;
  description: string;
  prompt: string;
}

/** Request body for PUT /skills/{id} */
export interface UpdateSkillRequest {
  name?: string;
  description?: string;
  prompt?: string;
}
