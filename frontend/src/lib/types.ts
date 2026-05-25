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
  /** In-memory preview result, if one exists for this agent. Not persisted. */
  preview?: PreviewResult;
}

// ── Preview / Apply (§8.21) ───────────────────────────────────────────────────

/**
 * Result from POST /sessions/{id}/agents/{agent_id}/preview.
 * Holds the agent's unpersisted output and the opaque preview_id token
 * used to guard concurrent apply calls.
 */
export interface PreviewResult {
  session_id: string;
  agent_id: string;
  preview_id: string;
  output: CanonicalState;
  created_at: string;
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

/**
 * One agent slot binding in a session — mirrors backend session.SessionAgent.
 * Populated on GET /sessions/{id} (not list). Contains only the binding info
 * (no agent name/model); cross-reference with the agent registry for full details.
 */
export interface SessionAgentSlot {
  session_id: string;
  agent_id: string;
  position: number;
  role: string;
}

export interface Session {
  id: string;
  idea: string;
  status: "active" | "running" | "converged" | "approved" | "failed";
  max_iterations: number;
  output_docs: string[];
  current_state: CanonicalState | null;
  created_at: string;
  updated_at: string;
  /** Ordered agent bindings — present on single GET, absent on list. */
  agents?: SessionAgentSlot[];
}

// ── API request / response shapes (§8.7) ─────────────────────────────────────

/** Body for POST /sessions */
export interface CreateSessionRequest {
  idea: string;
  agent_ids: string[]; // min 2 required
  max_iterations?: number;
  output_docs?: string[];
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

// ── Session list ──────────────────────────────────────────────────────────────

/**
 * Summary row returned by GET /sessions.
 * Idea is truncated to ≤ 120 chars by the backend service layer.
 * Confidence and current_iteration are extracted from current_state JSONB.
 */
export interface SessionListItem {
  id: string;
  idea: string; // ≤ 120 chars
  status: "active" | "converged" | "approved" | "failed";
  max_iterations: number;
  current_iteration: number;
  confidence: number; // [0.0, 1.0]
  agent_count: number;
  created_at: string;
  updated_at: string;
}

/** Envelope returned by GET /sessions. */
export interface ListSessionsResponse {
  sessions: SessionListItem[];
  total: number;
}

// ── Finalize response ─────────────────────────────────────────────────────────

/**
 * One generated output document returned by POST /sessions/{id}/finalize.
 */
export interface GeneratedDocument {
  filename: string;
  content: string;
  line_count: number;
}

/** Optional body for POST /sessions/{id}/finalize. */
export interface FinalizeRequest {
  output_docs?: string[];
}

/** Body for PATCH /sessions/{id}/output-docs. */
export interface UpdateOutputDocsRequest {
  output_docs: string[];
}

/**
 * Response from POST /sessions/{id}/finalize.
 * Documents is a map keyed by output doc key ("architecture", "roadmap", etc.).
 */
export interface FinalizeResponse {
  session_id: string;
  documents: Record<string, GeneratedDocument>;
  status: string;
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
