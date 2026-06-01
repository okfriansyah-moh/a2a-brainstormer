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
  import { createSSEClient } from "$lib/services/sse";
  import { API_BASE } from "$lib/services/api";
  import type { Agent, PreviewResult, SessionAgent } from "$lib/types";
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
      if (live === "done") return "done" as const;

      if ($sessionStore.loading) {
        if (hasSSEData) {
          // Infer "running" when all previous agents are confirmed "done".
          const allPrevDone = $sessionStore.agents
            .slice(0, i)
            .every((a) => $sessionStore.agentStatuses[a.id] === "done");
          if (allPrevDone) return "running" as const;
        } else {
          // No SSE data yet — best-guess: first agent is running.
          if (i === 0) return "running" as const;
        }
      }

      if (!$sessionStore.loading && currentIteration > 0)
        return "done" as const;
      return "waiting" as const;
    });
  })();

  /**
   * Short log text for the stage body (shown while running or done).
   * Emits a compact summary of counts without any raw LLM string dumps.
   */
  function stageOutputText(agent: SessionAgent): string {
    if (!agent.output) return "";
    const s = agent.output;
    const lines: string[] = [];
    if (s.execution_plan?.length) {
      lines.push(`Plan steps: ${s.execution_plan.length}`);
    }
    if (s.risks?.length) {
      lines.push(`Risks identified: ${s.risks.length}`);
    }
    if (s.open_questions?.length) {
      lines.push(`Open questions: ${s.open_questions.length}`);
    }
    if (s.assumptions?.length) {
      lines.push(`Assumptions: ${s.assumptions.length}`);
    }
    return lines.join("\n");
  }

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

  onMount(async () => {
    if (!sessionId) return;
    sessionStore.setLoading(true);
    loadError = "";
    // Track whether the server reports an iteration is actively running so we
    // can stay in loading mode and watch SSE instead of re-enabling the button.
    let iterationInFlight = false;
    try {
      const session = await getSession(sessionId);
      iterationInFlight = session.status === "running";
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

    // Open SSE stream for real-time agent progress events.
    sseClient = createSSEClient(
      `${API_BASE}/sessions/${sessionId}/events`,
      (evt) => {
        // Once convergence is confirmed, suppress ALL further engine progress
        // events (iteration.start / agent.started / iteration.complete) that
        // would reset the pipeline UI back to "running" — the engine has
        // already returned a final state and the user is on the convergence
        // screen waiting to finalize.
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
        // Track convergence from SSE so the page updates without a reload.
        if (evt.type === "iteration.complete") {
          const d = evt.data as { converged?: boolean } | null;
          if (d?.converged) {
            converged = true;
            // Force loading off — engine has returned; any further events are
            // residual / replayed and must not leave the UI "Running…".
            sessionStore.setLoading(false);
          }
        }
        if (evt.type === "session.finalized") {
          converged = true;
          sessionStore.setLoading(false);
        }
      },
    );

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
  });

  onDestroy(() => {
    sseClient?.close();
  });

  async function handleNextIteration() {
    if ($sessionStore.loading || !sessionId || converged) return;
    sessionStore.setLoading(true);
    plainIterPending = true;
    actionError = "";
    // Clear any local previews — a full pipeline pass supersedes them.
    previewMap = {};
    let iterInFlight = false;
    try {
      const result = await iterate(sessionId);
      sessionStore.updateState(result.state);
      converged = result.converged;
      // Engine returned — clear loading regardless of any in-flight SSE
      // events that may still be queued in the ring buffer.
      if (result.converged) sessionStore.setLoading(false);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        // Another iteration is already running. Stay in loading state and let
        // the SSE iteration.complete event clear it once the pass finishes.
        iterInFlight = true;
      } else {
        actionError = err instanceof Error ? err.message : "Iteration failed.";
      }
    } finally {
      // Always reset the pending flag — HTTP call is complete.
      plainIterPending = false;
      // Only clear loading if the engine isn't already in-flight. For the 409
      // case the SSE stream will fire iteration.complete which clears loading.
      if (!iterInFlight) {
        sessionStore.setLoading(false);
      }
    }
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
    actionError = "";
    showFeedback = false;
    feedbackText = "";
    // Optimistically un-converge so the UI immediately shows "running" state
    // rather than staying on the finalize prompt during the long iterate call.
    converged = false;
    sessionStore.setLoading(true);
    previewMap = {};
    let iterInFlight = false;
    try {
      const result = await iterate(sessionId, feedback);
      sessionStore.updateState(result.state);
      converged = result.converged;
      if (result.converged) sessionStore.setLoading(false);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        // The backend still has an iteration in flight for this session.
        // Restore the feedback text and re-show the panel so the user can
        // resubmit once the current pass completes. Do NOT clear loading —
        // the SSE iteration.complete event will do that.
        iterInFlight = true;
        feedbackText = feedback;
        showFeedback = true;
        converged = false;
        actionError =
          "An iteration is already running. Your feedback has been saved — submit again when the current pass completes.";
      } else {
        // If the backend rejects the feedback run, restore the converged state
        // so the UI returns to the finalize prompt. Prefer the response body
        // over the generic status message so the user sees the actual reason.
        converged = true;
        actionError =
          err instanceof ApiError && err.body
            ? err.body
            : err instanceof Error
              ? err.message
              : "Iteration with feedback failed.";
      }
    } finally {
      if (!iterInFlight) {
        sessionStore.setLoading(false);
      }
    }
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

  <div class="workspace">
    <!-- ── Pass summary bar ──────────────────────────────────────────── -->
    <div class="pass-header panel">
      <div>
        <div class="pass-label">
          Pipeline Pass
          <span>{currentIteration > 0 ? currentIteration : "—"}</span>
          {#if maxIterations > 0}
            / {maxIterations}
          {/if}
        </div>
        <div class="pass-sub">
          Sequential · {$sessionStore.agents.length} agents · Ordered by position
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
            output={stageOutputText(agent)}
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
            disabled={$sessionStore.loading}
          >
            {$sessionStore.loading ? "Running…" : "Run Next Iteration"}
          </button>
          <button
            class="btn-ghost"
            type="button"
            on:click={handleToggleFeedback}
            disabled={$sessionStore.loading}
          >
            Inject Feedback
          </button>
          <button
            class="btn-ghost"
            type="button"
            on:click={handleFinalize}
            disabled={$sessionStore.loading}
          >
            Finalize Session
          </button>
        {:else}
          <button class="btn-primary" type="button" on:click={handleFinalize}>
            Finalize Session →
          </button>
          <button
            class="btn-ghost"
            type="button"
            on:click={handleToggleFeedback}
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
            <span class="dot-live"></span> Running pipeline…
          </span>
        {:else if currentIteration > 0}
          <span style="color:var(--ink-500);font-size:0.8125rem;">
            Pass {currentIteration} complete · confidence {confidencePct}%
          </span>
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

  /* ── Responsive ── */
  @media (max-width: 900px) {
    .split {
      grid-template-columns: 1fr;
    }
  }
</style>
