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
    discardAgentPreview,
  } from "$lib/services/api";
  import { createSSEClient } from "$lib/services/sse";
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

  /** Confidence as 0–100 integer for ConfidenceBar. */
  $: confidencePct = Math.round(
    ($sessionStore.state?.metrics?.confidence ?? 0) * 100,
  );

  /** Current iteration number (0 before first iteration). */
  $: currentIteration = $sessionStore.state?.meta?.iteration ?? 0;

  /**
   * Per-stage status array driven by SSE agentStatuses.
   * Falls back to loading-based inference before any SSE events arrive.
   * Maps 'error' → 'waiting' since PipelineStage only accepts done/running/waiting.
   */
  $: stageStatuses = $sessionStore.agents.map((agent, i) => {
    const live = $sessionStore.agentStatuses[agent.id];
    if (live === "running") return "running" as const;
    if (live === "done") return "done" as const;
    // Fallback simulation when SSE data is absent.
    if (!$sessionStore.loading && currentIteration > 0) return "done" as const;
    if ($sessionStore.loading)
      return i === 0 ? ("running" as const) : ("waiting" as const);
    return "waiting" as const;
  });

  function stageOutputText(agent: SessionAgent): string {
    if (!agent.output) return "";
    const s = agent.output;
    const lines: string[] = [];
    if (s.architecture?.overview) {
      lines.push(`Architecture: ${s.architecture.overview}`);
    }
    if (s.execution_plan?.length) {
      lines.push(`Plan steps: ${s.execution_plan.length}`);
    }
    if (s.risks?.length) {
      lines.push(`Risks identified: ${s.risks.length}`);
    }
    if (s.open_questions?.length) {
      lines.push(`Open questions: ${s.open_questions.length}`);
    }
    return lines.join("\n") || JSON.stringify(s).slice(0, 300);
  }

  function stageSummaryText(agent: SessionAgent): string {
    if (!agent.output) return "";
    const ov = agent.output.architecture?.overview;
    if (ov) return ov.slice(0, 180);
    const firstStep = agent.output.execution_plan?.[0];
    if (firstStep) return firstStep.title;
    return "";
  }

  onMount(async () => {
    if (!sessionId) return;
    sessionStore.setLoading(true);
    loadError = "";
    try {
      const session = await getSession(sessionId);
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
      sessionStore.setLoading(false);
    }

    // Open SSE stream for real-time agent progress events.
    sseClient = createSSEClient(`/api/sessions/${sessionId}/events`, (evt) =>
      sessionStore.applyEvent(evt),
    );
  });

  onDestroy(() => {
    sseClient?.close();
  });

  async function handleNextIteration() {
    if ($sessionStore.loading || !sessionId || converged) return;
    sessionStore.setLoading(true);
    actionError = "";
    // Clear any local previews — a full pipeline pass supersedes them.
    previewMap = {};
    try {
      const result = await iterate(sessionId);
      sessionStore.updateState(result.state);
      converged = result.converged;
    } catch (err) {
      actionError = err instanceof Error ? err.message : "Iteration failed.";
    } finally {
      sessionStore.setLoading(false);
    }
  }

  async function handleFinalize() {
    if ($sessionStore.loading || !sessionId) return;
    await goto(`/session/${sessionId}/finalize`);
  }

  function handleToggleFeedback() {
    showFeedback = !showFeedback;
  }

  function handleInjectFeedback() {
    if (!feedbackText.trim()) return;
    // Feedback is surfaced in the UI for the next iterate call.
    // Full wiring is done in Task 15 integration.
    actionError = "";
    showFeedback = false;
    feedbackText = "";
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
      const { [agentId]: _removed, ...rest } = previewMap;
      previewMap = rest;
    } catch (err) {
      actionError =
        err instanceof Error ? err.message : "Apply preview failed.";
    } finally {
      previewRunningMap = { ...previewRunningMap, [agentId]: false };
    }
  }

  async function handleDiscardPreview(agentId: string) {
    if (!sessionId) return;
    try {
      await discardAgentPreview(sessionId, agentId);
    } catch {
      // Discard is best-effort — ignore errors.
    }
    const { [agentId]: _removed, ...rest } = previewMap;
    previewMap = rest;
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
    <nav class="topbar-nav">
      <a
        href="/"
        class="topbar-link"
        on:click={(e) => {
          e.preventDefault();
          goto("/");
        }}>← New Session</a
      >
      <a
        href="/settings"
        class="topbar-link"
        on:click={(e) => {
          e.preventDefault();
          goto("/settings");
        }}>⚙ Settings</a
      >
    </nav>
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
        <a
          href="/history"
          class="topbar-link"
          on:click={(e) => {
            e.preventDefault();
            goto("/history");
          }}>← Sessions</a
        >
        <ConfidenceBar
          value={confidencePct}
          animating={$sessionStore.loading}
        />
      </div>
    </div>

    <!-- ── Sequential pipeline ──────────────────────────────────────── -->
    {#if $sessionStore.agents.length > 0}
      <div class="panel pipeline">
        {#each $sessionStore.agents as agent, i (agent.id)}
          <PipelineStage
            {agent}
            position={i + 1}
            status={stageStatuses[i] ?? "waiting"}
            output={stageOutputText(agent)}
            summary={stageSummaryText(agent)}
            pipelineRunning={$sessionStore.loading}
            previewRunning={previewRunningMap[agent.id] ?? false}
            preview={previewMap[agent.id]}
            onPreview={() => handlePreviewAgent(agent.id)}
            onApply={() => handleApplyPreview(agent.id)}
          />
          {#if i < $sessionStore.agents.length - 1}
            <div
              class="stage-connector"
              class:stage-connector-dim={stageStatuses[i] === "waiting" ||
                stageStatuses[i + 1] === "waiting"}
            ></div>
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
    <div class="panel run-bar">
      <div class="run-left">
        <button
          class="btn-primary"
          type="button"
          on:click={handleNextIteration}
          disabled={$sessionStore.loading || converged}
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
  .pipeline {
    padding: 0;
    overflow: hidden;
  }

  .stage-connector {
    height: 1px;
    background: var(--line);
    margin: 0 18px;
  }

  .stage-connector-dim {
    background: var(--bg-1);
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
