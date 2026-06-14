<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { get } from "svelte/store";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import { sessionStore } from "$lib/stores/sessionStore";
  import { agentRegistryStore } from "$lib/stores/agentRegistryStore";
  import PipelineStage from "$lib/components/PipelineStage.svelte";
  import ConfidenceBar from "$lib/components/ConfidenceBar.svelte";
  import CanonicalStatePanel from "$lib/components/CanonicalStatePanel.svelte";
  import RiskBoard from "$lib/components/RiskBoard.svelte";
  import {
    getSession,
    getAgents,
    iterate,
    previewAgent,
    applyAgentPreview,
    ApiError,
  } from "$lib/services/api";
  import { createSSEClient, type SSEClientOptions } from "$lib/services/sse";
  import { API_BASE } from "$lib/services/api";
  import type { Agent, PreviewResult, SessionAgent, AgentPassContribution } from "$lib/types";
  import type { SSEClient } from "$lib/services/sse";

  /** True when the backend signals the session has converged. */
  let converged = false;

  /** Max iterations for the progress label — loaded with the session. */
  let maxIterations = 0;

  let loadError = "";
  let actionError = "";

  /** Active SSE client — closed on component destroy. */
  let sseClient: SSEClient | null = null;

  /** Controls visibility of the feedback textarea. */
  let showFeedback = false;
  let feedbackText = "";

  /**
   * True while a plain (no-feedback) iterate HTTP call is still in-flight.
   * SSE may clear loading=false before the HTTP response arrives, creating
   * a window where handleInjectFeedback incorrectly thinks it can run. This
   * flag tracks the full HTTP lifecycle so feedback is never silently dropped.
   */
  let plainIterPending = false;

  // ─── Iterate retry config ──────────────────────────────────────────────────
  // When the iterate HTTP call results in a TypeError (browser dropped the
  // long-running connection while the pipeline was still processing), we poll
  // getSession() in the background to confirm the server received the request.
  // Both values can be raised without UI changes — interval in ms, max attempts.
  const ITER_RETRY_INTERVAL_MS = 8_000; // poll every 8 s
  const ITER_RETRY_MAX = 30; // 30 × 8 s ≈ 4 min

  let iterRetryCount = 0;
  let iterRetryActive = false;
  let iterRetryTimer: ReturnType<typeof setTimeout> | null = null;

  function clearIterRetry() {
    if (iterRetryTimer !== null) {
      clearTimeout(iterRetryTimer);
      iterRetryTimer = null;
    }
    iterRetryActive = false;
    iterRetryCount = 0;
  }

  /**
   * Starts a background polling loop that calls getSession() every
   * ITER_RETRY_INTERVAL_MS. Behaviour per server status:
   * - "running"            → SSE will fire; keep retrying.
   * - "converged"/"approved" → apply state inline (SSE may have been missed).
   * - other / fetch error  → keep retrying until ITER_RETRY_MAX.
   * When retries are exhausted, surfaces a retryable actionError and restores
   * the converged flag to fallbackConverged.
   */
  function startIterRetry(sid: string, fallbackConverged: boolean) {
    clearIterRetry();
    iterRetryActive = true;

    function tick() {
      iterRetryTimer = setTimeout(async () => {
        iterRetryCount += 1;
        let resolved = false;
        try {
          const sess = await getSession(sid);
          if (sess.status === "converged" || sess.status === "approved") {
            // Pipeline completed but SSE missed the final event — apply inline.
            if (sess.current_state)
              sessionStore.updateState(sess.current_state);
            converged = true;
            resolved = true;
            clearIterRetry();
            sessionStore.setLoading(false);
          }
          // "running" → SSE is watching; keep retrying.
          // "active"/"failed" → server never received it or hard failure; keep retrying.
        } catch {
          // Network still unreachable — keep retrying until exhausted.
        }

        if (!resolved) {
          if (iterRetryCount < ITER_RETRY_MAX) {
            tick();
          } else {
            clearIterRetry();
            converged = fallbackConverged;
            sessionStore.setLoading(false);
            actionError =
              'Lost connection to the server. The pipeline may still be running — click "Run Next Iteration" to retry.';
          }
        }
      }, ITER_RETRY_INTERVAL_MS);
    }

    tick();
  }

  /** agentId → current phase detail string, cleared when agent.complete arrives. */
  let agentPhaseDetail: Record<string, string> = {};

  /** agentId → accumulated LLM token stream, cleared when agent.complete arrives. */
  let agentTokenBuffers: Record<string, string> = {};
  let agentErrorMessages: Record<string, string> = {};

  /** agentId → contributions from each completed pipeline pass. */
  let agentPassHistory: Record<string, AgentPassContribution[]> = {};

  /** Live pipeline pass from SSE iteration.start (accurate while a pass runs). */
  let pipelinePassNumber = 0;

  /** Agent currently executing (from agent.started SSE). */
  let activeAgentId: string | null = null;

  /** Shown briefly between iteration.complete and the next agent.started. */
  let passTransitionMessage = "";
  let awaitingPassStart = false;

  /** Wall-clock tick for elapsed-time labels while an agent runs. */
  let activityClock = Date.now();
  let activityClockTimer: ReturnType<typeof setInterval> | null = null;
  let agentActivityStartedAt: number | null = null;

  /**
   * Map of agentId → true while a preview dispatch is in flight for that agent.
   * Used to disable per-agent buttons during the request.
   */
  let previewRunningMap: Record<string, boolean> = {};

  /**
   * Map of agentId → PreviewResult for previews that have been fetched but
   * not yet applied. Displayed as the "Preview — not committed" banner.
   */
  let previewMap: Record<string, PreviewResult> = {};

  $: sessionId = $page.params.id;

  /**
   * Confidence as 0–100 integer for ConfidenceBar.
   * Guard against LLM agents that return confidence already on a 0-100 scale
   * (e.g. 95) instead of the canonical 0-1 scale — multiply would give 9500%.
   */
  $: confidencePct = (() => {
    const raw = $sessionStore.state?.metrics?.confidence ?? 0;
    return Math.min(100, Math.round(raw > 1 ? raw : raw * 100));
  })();

  /** Current iteration number (0 before first iteration). */
  $: currentIteration = $sessionStore.state?.meta?.iteration ?? 0;

  /**
   * Pass label shown in headers, banners, and the run bar.
   * Idle: mirror persisted meta.iteration (same source as passIdleSummary).
   * Loading: show the in-flight pass from SSE, never below completed + 1.
   */
  $: displayPass = $sessionStore.loading
    ? Math.max(pipelinePassNumber, currentIteration + 1)
    : currentIteration;

  /** Monotonic live pass counter from SSE — never regress to a completed pass. */
  function syncLivePass(iteration: number | undefined) {
    if (!iteration || iteration <= 0) return;
    pipelinePassNumber = Math.max(pipelinePassNumber, iteration);
  }

  /** Align the live counter with persisted meta when a pass finishes or page idles. */
  function snapPipelinePassToState() {
    const completed = get(sessionStore).state?.meta?.iteration ?? 0;
    if (completed > 0) {
      pipelinePassNumber = completed;
    }
  }

  /** Shared preflight for normal iteration and feedback-injection runs. */
  function beginPipelinePass() {
    const nextPass = (get(sessionStore).state?.meta?.iteration ?? 0) + 1;
    syncLivePass(nextPass);
    sessionStore.setLoading(true);
    plainIterPending = true;
    actionError = "";
    previewMap = {};
  }

  $: activeAgent = activeAgentId
    ? ($sessionStore.agents.find((a) => a.id === activeAgentId) ?? null)
    : null;

  function agentCompletedPass(agentId: string, pass: number): boolean {
    if (pass <= 0) return false;
    return (agentPassHistory[agentId] ?? []).some((p) => p.iteration === pass);
  }

  /** True when live SSE says "done" for the current in-flight pass (not a stale badge). */
  function agentDoneForCurrentPass(agentId: string): boolean {
    const live = $sessionStore.agentStatuses[agentId];
    if (live !== "done") return false;
    if ($sessionStore.loading && displayPass > 0) {
      return agentCompletedPass(agentId, displayPass);
    }
    return true;
  }

  function formatRole(role: string): string {
    return role.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
  }

  function formatElapsed(ms: number): string {
    const totalSec = Math.max(0, Math.floor(ms / 1000));
    const min = Math.floor(totalSec / 60);
    const sec = totalSec % 60;
    if (min > 0) return `${min}m ${sec}s`;
    return `${sec}s`;
  }

  $: passIdleSummary = (() => {
    if ($sessionStore.loading || converged || currentIteration <= 0) return "";
    const next = currentIteration + 1;
    if (maxIterations > 0 && next > maxIterations) {
      return `Pass ${currentIteration} complete · max passes reached`;
    }
    return `Pass ${currentIteration} complete · ${confidencePct}% confidence · ready for pass ${next}`;
  })();

  $: pipelineActivityTitle = (() => {
    if (!$sessionStore.loading || converged) return "";
    if (passTransitionMessage) return passTransitionMessage;
    const pass = displayPass > 0 ? displayPass : 1;
    if (activeAgent) return `Pass ${pass} · ${activeAgent.name}`;
    const runningIdx = stageStatuses.findIndex((s) => s === "running");
    if (runningIdx >= 0) {
      return `Pass ${pass} · ${$sessionStore.agents[runningIdx].name}`;
    }
    return `Pass ${pass} in progress`;
  })();

  $: pipelineActivityDetail = (() => {
    if (!$sessionStore.loading || converged || passTransitionMessage) return "";
    if (activeAgent) {
      const detail = agentPhaseDetail[activeAgent.id];
      const elapsed =
        agentActivityStartedAt != null
          ? formatElapsed(activityClock - agentActivityStartedAt)
          : "";
      if (detail && elapsed) return `${detail} · ${elapsed}`;
      if (detail) return detail;
      if (elapsed) {
        return `${formatRole(activeAgent.role)} · thinking · ${elapsed}`;
      }
      return `${formatRole(activeAgent.role)} · waiting for model output…`;
    }
    const runningIdx = stageStatuses.findIndex((s) => s === "running");
    if (runningIdx >= 0) {
      return `${formatRole($sessionStore.agents[runningIdx].role)} · preparing…`;
    }
    return "Advancing to the next agent in sequence…";
  })();

  $: runningButtonLabel = (() => {
    const pass = displayPass > 0 ? displayPass : 1;
    return `Running pass ${pass}…`;
  })();

  $: runStatusPrimary = (() => {
    if (!$sessionStore.loading) return "";
    const pass = displayPass > 0 ? displayPass : 1;
    return `Pass ${pass} in progress`;
  })();

  $: runStatusSecondary = (() => {
    if (!$sessionStore.loading) return "";
    if (passTransitionMessage) return passTransitionMessage;
    if (activeAgent) {
      const detail = agentPhaseDetail[activeAgent.id];
      return detail || `${activeAgent.name} is working`;
    }
    const runningIdx = stageStatuses.findIndex((s) => s === "running");
    if (runningIdx >= 0) {
      return `${$sessionStore.agents[runningIdx].name} is running`;
    }
    return "Waiting for the next agent";
  })();

  function recordPassContribution(
    agentId: string,
    iteration: number,
    agent: SessionAgent,
  ) {
    const { headline, bullets } = stageSummary(agent);
    const prev = agentPassHistory[agentId] ?? [];
    const next = [
      ...prev.filter((p) => p.iteration !== iteration),
      { iteration, headline, bullets },
    ].sort((a, b) => a.iteration - b.iteration);
    agentPassHistory = { ...agentPassHistory, [agentId]: next };
  }

  /**
   * Per-stage status array driven by SSE agentStatuses.
   *
   * Primary path: use the live SSE status if present ("running" or "done").
   *
   * Inference path (when loading=true and SSE data is available but the
   * "agent.started" event for the current agent has not arrived yet — e.g.
   * due to a brief reconnect during a long LLM call):
   *   - "running" is inferred for the first agent in the ordered list whose
   *     every preceding agent is already "done".  This correctly advances the
   *     highlighted stage as agents complete without requiring a perfectly
   *     gapless SSE stream.
   *
   * Fallback path (no SSE data at all): first agent shown as "running".
   * Post-iteration fallback: all agents shown as "done" once loading clears.
   */
  $: stageStatuses = (() => {
    const hasSSEData = Object.keys($sessionStore.agentStatuses).length > 0;
    return $sessionStore.agents.map((agent, i) => {
      const live = $sessionStore.agentStatuses[agent.id];
      if (live === "running") return "running" as const;
      if (live === "error") return "error" as const;
      if (agentDoneForCurrentPass(agent.id)) return "done" as const;

      if ($sessionStore.loading) {
        if (hasSSEData) {
          // Infer "running" when all previous agents finished this pass.
          const allPrevDone = $sessionStore.agents
            .slice(0, i)
            .every((a) => agentDoneForCurrentPass(a.id));
          if (allPrevDone) return "running" as const;
        } else if (i === 0) {
          return "running" as const;
        }
      }

      if (!$sessionStore.loading && currentIteration > 0)
        return "done" as const;
      return "waiting" as const;
    });
  })();

  /**
   * Pick the first non-empty string from a list of candidates. Handles raw
   * LLM payloads where the same field may appear under different keys
   * (title, name, phase_name, step, action, etc.).
   */
  function firstString(...candidates: unknown[]): string {
    for (const c of candidates) {
      if (typeof c === "string") {
        const t = c.trim();
        if (t.length > 0) return t;
      }
    }
    return "";
  }

  /** Join a list of strings with commas and a final "and". */
  function joinHuman(parts: string[]): string {
    if (parts.length === 0) return "";
    if (parts.length === 1) return parts[0];
    if (parts.length === 2) return `${parts[0]} and ${parts[1]}`;
    return `${parts.slice(0, -1).join(", ")}, and ${parts[parts.length - 1]}`;
  }

  /** Truncate a label to keep bullet lines readable. */
  function clip(s: string, max: number): string {
    return s.length > max ? s.slice(0, max - 1).trimEnd() + "…" : s;
  }

  /**
   * Build a human-readable contribution summary for the "Contribution:" block.
   * Returns a prose headline (always populated when the agent ran) plus an
   * optional list of bullets. The headline reads like a docs changelog entry.
   */
  function stageSummary(agent: SessionAgent): {
    headline: string;
    bullets: string[];
  } {
    if (!agent.output) return { headline: "", bullets: [] };
    const s = agent.output;

    const planCount = s.execution_plan?.length ?? 0;
    const riskCount = s.risks?.length ?? 0;
    const assumptionCount = s.assumptions?.length ?? 0;
    const questionCount = s.open_questions?.length ?? 0;
    const hasArchitecture = !!(
      s.architecture?.overview ||
      s.architecture?.components?.length ||
      s.architecture?.decisions?.length
    );

    // Build prose phrases describing what changed in this pass.
    const parts: string[] = [];
    if (planCount > 0) {
      parts.push(`a ${planCount}-step execution plan`);
    }
    if (hasArchitecture) parts.push("architecture notes");
    if (riskCount > 0) {
      parts.push(`${riskCount} risk${riskCount === 1 ? "" : "s"}`);
    }
    if (assumptionCount > 0) {
      parts.push(
        `${assumptionCount} assumption${assumptionCount === 1 ? "" : "s"}`,
      );
    }
    if (questionCount > 0) {
      parts.push(
        `${questionCount} open question${questionCount === 1 ? "" : "s"}`,
      );
    }

    const role = agent.role.toLowerCase();
    let verb = "Contributed";
    if (role.includes("review") || role.includes("critic")) {
      verb = "Reviewed the canonical state and added";
    } else if (role.includes("synth") || role.includes("merge")) {
      verb = "Synthesised the pass with";
    } else if (role.includes("build") || role.includes("architect")) {
      verb = "Drafted";
    }

    const headline =
      parts.length === 0
        ? `${agent.name} ran but produced no new structured findings this pass.`
        : `${verb} ${joinHuman(parts)} to the canonical state.`;

    // Pick the most informative bullet list: plan steps first, then risks.
    const bullets: string[] = [];
    if (planCount > 0) {
      s.execution_plan.slice(0, 5).forEach((step, idx) => {
        const raw = step as unknown as Record<string, unknown>;
        const label = firstString(
          step.title,
          raw["name"],
          raw["phase_name"],
          raw["step"],
          raw["action"],
          raw["task"],
          step.description,
        );
        bullets.push(clip(label || `Step ${idx + 1}`, 110));
      });
      if (planCount > 5) bullets.push(`+${planCount - 5} more phases`);
    } else if (riskCount > 0) {
      s.risks.slice(0, 5).forEach((r) => {
        const raw = r as unknown as Record<string, unknown>;
        const label = firstString(r.title, raw["text"], r.description);
        bullets.push(
          `[${r.severity}] ${clip(label || "(unlabelled risk)", 90)}`,
        );
      });
      if (riskCount > 5) bullets.push(`+${riskCount - 5} more`);
    } else if (hasArchitecture) {
      const ov = s.architecture.overview?.trim() ?? "";
      // Skip raw Go map-serialized strings (LLM formatting artefacts).
      if (ov && !ov.startsWith("[map[") && !ov.startsWith("map[")) {
        bullets.push(clip(ov, 240));
      }
    }

    return { headline, bullets };
  }

  function sessionSSEOptions(): SSEClientOptions {
    return {
      beforeReconnect: async () => {
        if (!sessionId) return false;
        try {
          await getSession(sessionId);
          return true;
        } catch (err) {
          if (err instanceof ApiError && err.status === 404) return false;
          return true;
        }
      },
    };
  }

  function connectSessionSSE() {
    if (!sessionId || sseClient) return;
    sseClient = createSSEClient(
      `${API_BASE}/sessions/${sessionId}/events`,
      (evt) => {
        if (converged) {
          if (
            evt.type === "iteration.start" ||
            evt.type === "agent.started" ||
            evt.type === "agent.complete" ||
            evt.type === "agent.error" ||
            evt.type === "iteration.complete"
          ) {
            return;
          }
        }
        sessionStore.applyEvent(evt);

        if (evt.type === "iteration.start") {
          const d = evt.data as { iteration?: number } | null;
          syncLivePass(d?.iteration);
          activeAgentId = null;
          agentTokenBuffers = {};
          agentPhaseDetail = {};
          agentErrorMessages = {};
        }

        if (evt.type === "agent.started") {
          const d = evt.data as { agent_id?: string; iteration?: number } | null;
          syncLivePass(d?.iteration);
          if (d?.agent_id) {
            activeAgentId = d.agent_id;
            agentActivityStartedAt = Date.now();
            activityClock = agentActivityStartedAt;
            if (awaitingPassStart) {
              passTransitionMessage = "";
              awaitingPassStart = false;
            }
            const updatedErrors = { ...agentErrorMessages };
            delete updatedErrors[d.agent_id];
            agentErrorMessages = updatedErrors;
            const updatedTokens = { ...agentTokenBuffers };
            delete updatedTokens[d.agent_id];
            agentTokenBuffers = updatedTokens;
          }
        }

        if (evt.type === "agent.phase") {
          const d = evt.data as {
            agent_id?: string;
            detail?: string;
            iteration?: number;
          } | null;
          syncLivePass(d?.iteration);
          if (d?.agent_id && d?.detail) {
            agentPhaseDetail = { ...agentPhaseDetail, [d.agent_id]: d.detail };
          }
        }
        if (evt.type === "agent.error") {
          const d = evt.data as { agent_id?: string; error?: string } | null;
          if (d?.agent_id && d?.error) {
            agentErrorMessages[d.agent_id] = d.error;
            agentErrorMessages = agentErrorMessages;
          }
        }
        if (evt.type === "agent.token") {
          const d = evt.data as { agent_id?: string; token?: string } | null;
          if (d?.agent_id && d?.token) {
            agentTokenBuffers[d.agent_id] =
              (agentTokenBuffers[d.agent_id] ?? "") + d.token;
            agentTokenBuffers = agentTokenBuffers;
          }
        }
        if (evt.type === "agent.complete") {
          const d = evt.data as {
            agent_id?: string;
            iteration?: number;
          } | null;
          syncLivePass(d?.iteration);
          if (d?.agent_id) {
            const iter = d.iteration ?? displayPass;
            const agent = get(sessionStore).agents.find(
              (a) => a.id === d.agent_id,
            );
            if (agent?.output && iter > 0) {
              recordPassContribution(d.agent_id, iter, agent);
            }
            if (d.agent_id === activeAgentId) {
              activeAgentId = null;
            }
            const updatedPhase = { ...agentPhaseDetail };
            delete updatedPhase[d.agent_id];
            agentPhaseDetail = updatedPhase;
            const updatedTokens = { ...agentTokenBuffers };
            delete updatedTokens[d.agent_id];
            agentTokenBuffers = updatedTokens;
          }
        }

        if (evt.type === "iteration.complete") {
          const d = evt.data as { converged?: boolean; iteration?: number } | null;
          if (d?.iteration) {
            syncLivePass(d.iteration);
          }
          activeAgentId = null;
          agentActivityStartedAt = null;
          passTransitionMessage = "";
          awaitingPassStart = false;
          if (d?.converged) {
            converged = true;
          }
        }
        if (evt.type === "session.finalized") {
          converged = true;
          sessionStore.setLoading(false);
        }
      },
      undefined,
      sessionSSEOptions(),
    );
  }

  onMount(async () => {
    activityClockTimer = setInterval(() => {
      activityClock = Date.now();
    }, 1000);

    if (!sessionId) return;
    loadError = "";
    const shouldAutoStart = $page.url.searchParams.get("autostart") === "1";
    // Track whether the server reports an iteration is actively running so we
    // can stay in loading mode and watch SSE instead of re-enabling the button.
    let iterationInFlight = false;
    let loadedIteration = 0;
    try {
      const session = await getSession(sessionId);
      iterationInFlight = session.status === "running";
      if (iterationInFlight) {
        sessionStore.setLoading(true);
      }
      loadedIteration = session.current_state?.meta?.iteration ?? 0;
      sessionStore.setSession(session.id, session.idea);
      maxIterations = session.max_iterations;
      if (session.current_state) {
        sessionStore.updateState(session.current_state);
        converged =
          session.status === "converged" || session.status === "approved";
      }
      if (session.current_state?.meta?.agents) {
        const agentsFromMeta: SessionAgent[] =
          session.current_state.meta.agents.map((a) => ({
            id: a.agent_id,
            name: a.name,
            role: a.role,
            provider: a.provider,
            model: a.model,
            skills: a.skills,
            output: undefined,
          }));
        sessionStore.setAgents(agentsFromMeta);
      } else if (session.agents && session.agents.length > 0) {
        // No iteration run yet — build agent display from session bindings +
        // the agent registry (so names/provider/model are shown immediately).
        let registry = get(agentRegistryStore).agents;
        if (registry.length === 0) {
          const loaded = await getAgents();
          agentRegistryStore.setAgents(loaded);
          registry = loaded;
        }
        const byId = new Map<string, Agent>(registry.map((a) => [a.id, a]));
        const agentsFromSlots: SessionAgent[] = session.agents.map((slot) => {
          const full = byId.get(slot.agent_id);
          return {
            id: slot.agent_id,
            name: full?.name ?? slot.agent_id,
            role: slot.role,
            provider: full?.llm_config.provider ?? "unknown",
            model: full?.llm_config.model ?? "unknown",
            skills: full?.skills?.map((s) => s.name) ?? [],
            output: undefined,
          };
        });
        sessionStore.setAgents(agentsFromSlots);
      }
      connectSessionSSE();
      if (session.current_state?.meta?.iteration) {
        const persisted = session.current_state.meta.iteration;
        // While the server is still running, meta.iteration is the last *completed*
        // pass — the live pass is usually one ahead.
        if (iterationInFlight) {
          syncLivePass(persisted + 1);
        } else {
          syncLivePass(persisted);
        }
      }
    } catch (err) {
      loadError =
        err instanceof Error ? err.message : "Failed to load session.";
    } finally {
      // Keep loading=true when an iteration is in-flight — the SSE
      // iteration.complete event will clear it once the pass finishes.
      if (!iterationInFlight) {
        sessionStore.setLoading(false);
      }
    }

    // Fallback: if the backend completed the iteration BEFORE we connected to
    // SSE (page reload after a run), the iteration.complete event was already
    // fired and will not be replayed. In that case loading stays permanently
    // true. Re-fetch the session after a short delay; if the status is no
    // longer "running" we know the iteration finished and can sync state.
    if (iterationInFlight) {
      setTimeout(async () => {
        if (!get(sessionStore).loading) return; // SSE already resolved it
        try {
          const refreshed = await getSession(sessionId);
          if (refreshed.status !== "running") {
            if (refreshed.current_state) {
              sessionStore.updateState(refreshed.current_state);
            }
            sessionStore.setLoading(false);
            if (
              refreshed.status === "converged" ||
              refreshed.status === "approved"
            ) {
              converged = true;
            }
          }
        } catch {
          // Best-effort: clear loading so the UI isn't permanently stuck.
          sessionStore.setLoading(false);
        }
      }, 5000);
    }

    if (
      shouldAutoStart &&
      !loadError &&
      !converged &&
      !iterationInFlight &&
      loadedIteration === 0
    ) {
      const url = new URL(window.location.href);
      url.searchParams.delete("autostart");
      const nextPath = url.pathname + (url.search ? url.search : "");
      history.replaceState({}, "", nextPath);
      await handleNextIteration();
    }
  });

  onDestroy(() => {
    if (activityClockTimer !== null) {
      clearInterval(activityClockTimer);
      activityClockTimer = null;
    }
    sseClient?.close();
    clearIterRetry();
  });

  async function runPipelinePass(userFeedback?: string) {
    if ($sessionStore.loading || !sessionId) return;
    if (converged && !userFeedback) return;

    const previousConverged = converged;
    const feedback = userFeedback?.trim();
    if (userFeedback !== undefined && !feedback) return;

    if (feedback) {
      converged = false;
    }

    beginPipelinePass();

    let iterInFlight = false;
    try {
      const result = await iterate(sessionId, feedback);
      sessionStore.updateState(result.state);
      converged = result.converged;
      syncLivePass(result.state.meta?.iteration);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        iterInFlight = true;
        if (feedback) {
          feedbackText = feedback;
          showFeedback = true;
          actionError =
            "An iteration is already running. Your feedback has been saved — submit again when the current pass completes.";
        }
      } else if (
        err instanceof TypeError &&
        /failed to fetch|networkerror|fetch/i.test(err.message)
      ) {
        iterInFlight = true;
        startIterRetry(sessionId, feedback ? previousConverged : false);
      } else if (feedback) {
        converged = previousConverged;
        actionError =
          err instanceof ApiError && err.body
            ? err.body
            : err instanceof Error
              ? err.message
              : "Iteration with feedback failed.";
      } else {
        actionError = err instanceof Error ? err.message : "Iteration failed.";
      }
    } finally {
      plainIterPending = false;
      if (!iterInFlight) {
        sessionStore.setLoading(false);
        snapPipelinePassToState();
      }
    }
  }

  async function handleNextIteration() {
    await runPipelinePass();
  }

  async function handleFinalize() {
    if ($sessionStore.loading || !sessionId) return;
    await goto(`/session/${sessionId}/finalize`);
  }

  function handleToggleFeedback() {
    showFeedback = !showFeedback;
  }

  async function handleInjectFeedback() {
    if (!feedbackText.trim()) return;

    // Guard: a plain iterate is still awaiting its HTTP response even though
    // SSE may have cleared loading=false (race window). Block the feedback
    // iterate to avoid conflicting state and show a clear message instead of
    // silently dropping the feedback.
    if (plainIterPending || !sessionId) {
      if (plainIterPending) {
        actionError =
          "The previous iteration is still completing. Please wait a moment before adding feedback.";
      }
      return;
    }

    if ($sessionStore.loading) return;

    const feedback = feedbackText.trim();
    showFeedback = false;
    feedbackText = "";
    await runPipelinePass(feedback);
  }

  async function handlePreviewAgent(agentId: string) {
    if (!sessionId || $sessionStore.loading || previewRunningMap[agentId])
      return;
    previewRunningMap = { ...previewRunningMap, [agentId]: true };
    actionError = "";
    try {
      const result = await previewAgent(sessionId, agentId);
      previewMap = { ...previewMap, [agentId]: result };
    } catch (err) {
      actionError =
        err instanceof Error ? err.message : "Preview dispatch failed.";
    } finally {
      previewRunningMap = { ...previewRunningMap, [agentId]: false };
    }
  }

  async function handleApplyPreview(agentId: string) {
    if (!sessionId || $sessionStore.loading || previewRunningMap[agentId])
      return;
    const existing = previewMap[agentId];
    if (!existing) return;
    previewRunningMap = { ...previewRunningMap, [agentId]: true };
    actionError = "";
    try {
      const newState = await applyAgentPreview(
        sessionId,
        agentId,
        existing.preview_id,
      );
      sessionStore.updateState(newState);
      // Clear the applied preview locally.
      const updated = { ...previewMap };
      delete updated[agentId];
      previewMap = updated;
    } catch (err) {
      actionError =
        err instanceof Error ? err.message : "Apply preview failed.";
    } finally {
      previewRunningMap = { ...previewRunningMap, [agentId]: false };
    }
  }
</script>

<div class="artboard">
  <!-- ── Topbar ────────────────────────────────────────────────────────── -->
  <div class="topbar session-topbar">
    <div>
      <div class="topbar-title">Session Workspace</div>
      {#if $sessionStore.idea}
        <div class="topbar-subtitle" title={$sessionStore.idea}>
          {$sessionStore.idea.length > 80
            ? $sessionStore.idea.slice(0, 77) + "…"
            : $sessionStore.idea}
        </div>
      {/if}
    </div>
  </div>

  <!-- ── Error banners ────────────────────────────────────────────────── -->
  {#if loadError}
    <div class="banner banner-error">{loadError}</div>
  {/if}
  {#if actionError}
    <div class="banner banner-warn">{actionError}</div>
  {/if}
  {#if iterRetryActive}
    <div class="banner banner-info">
      <span class="dot-live"></span>
      Checking server status… attempt {iterRetryCount}/{ITER_RETRY_MAX}
      <button class="btn-ghost btn-xs" type="button" on:click={clearIterRetry}>
        Cancel
      </button>
    </div>
  {/if}
  {#if $sessionStore.loading && !converged && pipelineActivityTitle}
    <div class="banner banner-activity" role="status" aria-live="polite">
      <span class="dot-live"></span>
      <div class="activity-copy">
        <div class="activity-title">{pipelineActivityTitle}</div>
        {#if pipelineActivityDetail}
          <div class="activity-detail">{pipelineActivityDetail}</div>
        {/if}
      </div>
    </div>
  {:else if !$sessionStore.loading && passIdleSummary && !converged}
    <div class="banner banner-pass-complete" role="status">
      <span class="chip-ok">✓</span>
      <span>{passIdleSummary}</span>
    </div>
  {/if}

  <div class="workspace">
    <!-- ── Pass summary bar ──────────────────────────────────────────── -->
    <div class="pass-header panel">
      <div>
        <div class="pass-label">
          Pipeline Pass
          <span>{displayPass > 0 ? displayPass : "—"}</span>
          {#if maxIterations > 0}
            / {maxIterations}
          {/if}
          {#if $sessionStore.loading && !converged}
            <span class="pass-live-tag">Live</span>
          {/if}
        </div>
        <div class="pass-sub">
          {#if $sessionStore.loading && !converged && pipelineActivityDetail}
            {pipelineActivityDetail}
          {:else}
            Sequential · {$sessionStore.agents.length} agents · Ordered by position
          {/if}
        </div>
      </div>
      <div class="pass-actions">
        <ConfidenceBar
          value={confidencePct}
          animating={$sessionStore.loading}
        />
      </div>
    </div>

    <!-- ── Sequential pipeline ──────────────────────────────────────── -->
    {#if $sessionStore.agents.length > 0}
      <div class="pipeline-list">
        {#each $sessionStore.agents as agent, i (agent.id)}
          <PipelineStage
            {agent}
            position={i + 1}
            status={stageStatuses[i] ?? "waiting"}
            passHistory={agentPassHistory[agent.id] ?? []}
            activePass={stageStatuses[i] === "running" ? displayPass : 0}
            streamingText={agentTokenBuffers[agent.id] ?? ""}
            phaseDetail={agentPhaseDetail[agent.id] ?? ""}
            errorMessage={agentErrorMessages[agent.id] ?? ""}
            summary={stageSummary(agent).headline}
            summaryBullets={stageSummary(agent).bullets}
            pipelineRunning={$sessionStore.loading}
            previewRunning={previewRunningMap[agent.id] ?? false}
            preview={previewMap[agent.id]}
            onPreview={() => handlePreviewAgent(agent.id)}
            onApply={() => handleApplyPreview(agent.id)}
          />
          {#if i < $sessionStore.agents.length - 1}
            <div class="stage-arrow" aria-hidden="true">
              <svg width="16" height="20" viewBox="0 0 16 20" fill="none">
                <path
                  d="M8 0 L8 14 M3 9 L8 14 L13 9"
                  stroke="var(--ink-300)"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </div>
          {/if}
        {/each}
      </div>
    {:else if !$sessionStore.loading && !loadError}
      <div class="panel no-agents">
        <p>No agents in this session. Run the first iteration to begin.</p>
      </div>
    {/if}

    <!-- ── Feedback panel (conditionally shown) ─────────────────────── -->
    {#if showFeedback}
      <div class="panel feedback-panel">
        <div class="feedback-label">Inject Feedback for Next Iteration</div>
        <textarea
          class="feedback-textarea"
          bind:value={feedbackText}
          placeholder="Describe what you want the agents to focus on or change in the next pass…"
          rows="4"
        ></textarea>
        <div class="feedback-actions">
          <button
            class="btn-primary"
            type="button"
            on:click={handleInjectFeedback}
            disabled={!feedbackText.trim()}
          >
            Queue Feedback
          </button>
          <button
            class="btn-ghost"
            type="button"
            on:click={handleToggleFeedback}
          >
            Cancel
          </button>
        </div>
      </div>
    {/if}

    <!-- ── Run bar ───────────────────────────────────────────────────── -->
    <!--
      Button states (single source of truth):
        converged=true  → Finalize is PRIMARY, Run Next Iteration is hidden
        loading=true    → Run Next Iteration shows "Running…" (disabled),
                          Finalize disabled (engine writing state)
        idle            → Run Next Iteration is PRIMARY, Finalize secondary
      Per-agent "Run This Agent" buttons stay enabled when converged so the
      user can experiment without re-running the whole pipeline.
    -->
    <div class="panel run-bar">
      <div class="run-left">
        {#if !converged}
          <button
            class="btn-primary"
            type="button"
            on:click={handleNextIteration}
            disabled={$sessionStore.loading || showFeedback}
          >
            {$sessionStore.loading ? runningButtonLabel : "Run Next Iteration"}
          </button>
          <button
            class="btn-ghost"
            type="button"
            on:click={handleToggleFeedback}
            disabled={$sessionStore.loading || showFeedback}
          >
            Inject Feedback
          </button>
          <button
            class="btn-ghost"
            type="button"
            on:click={handleFinalize}
            disabled={$sessionStore.loading || showFeedback}
          >
            Finalize Session
          </button>
        {:else}
          <button
            class="btn-primary"
            type="button"
            on:click={handleFinalize}
            disabled={$sessionStore.loading || showFeedback}
          >
            Finalize Session →
          </button>
          <button
            class="btn-ghost"
            type="button"
            on:click={handleToggleFeedback}
            disabled={$sessionStore.loading || showFeedback}
          >
            Inject Feedback
          </button>
        {/if}
      </div>
      <div class="run-status">
        {#if converged}
          <span class="chip-ok">✓ Converged — ready to finalize</span>
        {:else if $sessionStore.loading}
          <span class="chip-live">
            <span class="dot-live"></span>
            <span class="run-status-text">
              <span class="run-status-primary">{runStatusPrimary}</span>
              {#if runStatusSecondary}
                <span class="run-status-secondary">{runStatusSecondary}</span>
              {/if}
            </span>
          </span>
        {:else if currentIteration > 0}
          <span class="chip-ok">{passIdleSummary || `Pass ${currentIteration} complete · ${confidencePct}%`}</span>
        {:else}
          <span style="color:var(--ink-300);font-size:0.8125rem;">
            Not started
          </span>
        {/if}
      </div>
    </div>

    <!-- ── Bottom split: canonical state + risk board ───────────────── -->
    <div class="split">
      <div class="panel split-state">
        <div class="section-heading">Canonical State</div>
        <CanonicalStatePanel state={$sessionStore.state} />
      </div>
      <div class="panel split-risks">
        <div class="section-heading">Risk Board</div>
        <RiskBoard risks={$sessionStore.state?.risks ?? []} />
      </div>
    </div>
  </div>
</div>

<style>
  .session-topbar {
    border-radius: 18px 18px 0 0;
    padding: 0 28px;
  }

  .topbar-title {
    font-family: "Space Grotesk", sans-serif;
    font-weight: 700;
    font-size: 1rem;
    color: var(--ink-900);
  }

  .topbar-subtitle {
    font-size: 0.8125rem;
    color: var(--ink-500);
    margin-top: 1px;
    max-width: 560px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .workspace {
    padding: 20px 28px 28px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  /* ── Pass header ── */
  .pass-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 18px;
  }

  .pass-label {
    font-family: "Space Grotesk", sans-serif;
    font-weight: 700;
    font-size: 1rem;
  }

  .pass-sub {
    color: var(--ink-500);
    font-size: 0.75rem;
    margin-top: 3px;
  }

  .pass-actions {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  /* ── Pipeline ── */
  .pipeline-list {
    display: flex;
    flex-direction: column;
    gap: 0;
  }

  .stage-arrow {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 24px;
    flex-shrink: 0;
  }

  .no-agents {
    padding: 24px;
    color: var(--ink-300);
    font-size: 0.875rem;
    font-style: italic;
  }

  /* ── Feedback panel ── */
  .feedback-panel {
    padding: 16px 18px;
  }

  .feedback-label {
    font-weight: 600;
    font-size: 0.8125rem;
    color: var(--ink-700);
    margin-bottom: 8px;
  }

  .feedback-textarea {
    width: 100%;
    border: 1.5px solid var(--line);
    border-radius: 8px;
    padding: 10px 12px;
    font-size: 0.875rem;
    background: rgba(255, 255, 255, 0.6);
    color: var(--ink-900);
    resize: vertical;
    outline: none;
  }

  .feedback-textarea:focus {
    border-color: var(--accent);
  }

  .feedback-actions {
    display: flex;
    gap: 8px;
    margin-top: 10px;
  }

  /* ── Run bar ── */
  .run-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    gap: 12px;
    flex-wrap: wrap;
  }

  .run-left {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .run-status {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  /* ── Bottom split ── */
  .split {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: 14px;
  }

  .split-state {
    padding: 18px 20px;
  }

  .split-risks {
    padding: 18px 20px;
  }

  .section-heading {
    font-family: "Space Grotesk", sans-serif;
    font-weight: 600;
    font-size: 0.875rem;
    color: var(--ink-900);
    margin-bottom: 12px;
  }

  /* ── Banners ── */
  .banner {
    margin: 0 28px;
    border-radius: 8px;
    padding: 10px 14px;
    font-size: 0.875rem;
    margin-top: 12px;
  }

  .banner-error {
    background: var(--danger-bg);
    color: var(--danger);
    border: 1px solid var(--danger-line);
  }

  .banner-warn {
    background: var(--warn-bg);
    color: var(--warn);
    border: 1px solid var(--warn-line);
  }

  .banner-info {
    background: var(--accent-bg);
    color: var(--accent-2);
    border: 1px solid var(--accent-line);
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .banner-pass-complete {
    background: var(--ok-bg);
    color: var(--ok);
    border: 1px solid var(--ok-line);
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.875rem;
  }

  .banner-activity {
    background: var(--accent-bg);
    color: var(--accent-2);
    border: 1px solid var(--accent-line);
    display: flex;
    align-items: flex-start;
    gap: 10px;
  }

  .activity-copy {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .activity-title {
    font-weight: 600;
    font-size: 0.875rem;
    color: var(--ink-900);
  }

  .activity-detail {
    font-size: 0.8125rem;
    color: var(--ink-600);
  }

  .pass-live-tag {
    display: inline-flex;
    align-items: center;
    margin-left: 8px;
    padding: 2px 8px;
    border-radius: 999px;
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    background: var(--accent-bg);
    color: var(--accent-2);
    border: 1px solid var(--accent-line);
    vertical-align: middle;
  }

  .run-status-text {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 1px;
  }

  .run-status-primary {
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--accent-2);
  }

  .run-status-secondary {
    font-size: 0.75rem;
    color: var(--ink-500);
    max-width: 280px;
    text-align: right;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .btn-xs {
    padding: 2px 8px;
    font-size: 0.75rem;
    margin-left: auto;
  }

  /* ── Responsive ── */
  @media (max-width: 900px) {
    .split {
      grid-template-columns: 1fr;
    }
  }
</style>
