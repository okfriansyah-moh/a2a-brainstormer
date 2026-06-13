<script lang="ts">
  import { onDestroy } from "svelte";
  import type { PreviewResult, SessionAgent, AgentPassContribution } from "$lib/types";
  import {
    AGENT_LOADING_PHRASES,
    DEFAULT_LOADING_PHRASES,
  } from "$lib/loadingPhrases";
  import {
    presentStreamBuffer,
    waitingPresentation,
  } from "$lib/streamPresenter";

  /** The agent represented by this stage. */
  export let agent: SessionAgent;

  /** Stage position number (1-based). */
  export let position: number;

  /** Execution state of this stage. */
  export let status: "done" | "running" | "waiting" | "error" = "waiting";

  /** Live phase label from the backend (e.g. drafting architecture). */
  export let phaseDetail: string = "";

  /** Error message when status is error. */
  export let errorMessage: string = "";

  /** Completed contributions from earlier pipeline passes, oldest first. */
  export let passHistory: AgentPassContribution[] = [];

  /** Pipeline pass number shown while this agent is running (from SSE). */
  export let activePass = 0;

  /** Live token stream while the agent is running. */
  export let streamingText: string = "";

  /** Human-readable summary produced after completion. */
  export let summary: string = "";

  /**
   * Optional structured bullet list rendered under the summary headline.
   * Each item is shown as a real <li> for readability — avoids dumping a
   * newline-joined bullet string that looks system-generated.
   */
  export let summaryBullets: string[] = [];

  /**
   * Whether the global pipeline is currently running (disables per-agent buttons
   * while a full iterate pass or another preview is in flight).
   */
  export let pipelineRunning: boolean = false;

  /**
   * In-flight flag for this specific agent's preview dispatch.
   * The parent page sets this to true while awaiting the previewAgent() call.
   */
  export let previewRunning: boolean = false;

  /**
   * Stored preview result for this agent, if one exists.
   * Enables the Apply button and shows the preview banner.
   */
  export let preview: PreviewResult | undefined = undefined;

  /** Fired when the user clicks "Run This Agent". */
  export let onPreview: (() => void) | undefined = undefined;

  /** Fired when the user clicks "Apply". */
  export let onApply: (() => void) | undefined = undefined;

  $: roleCssClass = agent.role.replace(/_/g, "-").toLowerCase();
  $: badgeLabel = agent.role.replace(/_/g, " ").toUpperCase();

  $: previewOutputText = preview
    ? JSON.stringify(preview.output, null, 2).slice(0, 1500)
    : "";

  $: canPreview = !pipelineRunning && !previewRunning;
  $: canApply = !pipelineRunning && !previewRunning && !!preview;

  const ROTATE_MS = 2600;
  let rotateIndex = 0;
  let rotateTimer: ReturnType<typeof setInterval> | null = null;

  $: loadingPhrases =
    AGENT_LOADING_PHRASES[agent.role] ?? DEFAULT_LOADING_PHRASES;
  $: showRotating =
    status === "running" && !streamingText && !phaseDetail;
  $: statusLine =
    phaseDetail ||
    (showRotating ? loadingPhrases[rotateIndex % loadingPhrases.length] : "");

  $: agentInitials = agent.name
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? "")
    .join("");

  $: chatPresent =
    status === "running"
      ? streamingText
        ? presentStreamBuffer(streamingText, statusLine, agent.role)
        : waitingPresentation(statusLine, agent.role)
      : null;

  $: {
    if (showRotating) {
      if (!rotateTimer) {
        rotateIndex = 0;
        rotateTimer = setInterval(() => {
          rotateIndex = (rotateIndex + 1) % loadingPhrases.length;
        }, ROTATE_MS);
      }
    } else if (rotateTimer) {
      clearInterval(rotateTimer);
      rotateTimer = null;
    }
  }

  onDestroy(() => {
    if (rotateTimer) {
      clearInterval(rotateTimer);
      rotateTimer = null;
    }
  });
</script>

<div class="stage stage-{status}" class:stage-error={status === "error"} role="region" aria-label={agent.name}>
  <div class="stage-header">
    <div class="stage-left">
      <span class="stage-num">{position}</span>
      <div>
        <div class="stage-name">
          {agent.name}
          <span class="badge-{roleCssClass}">{badgeLabel}</span>
        </div>
        <div class="stage-model">{agent.provider} / {agent.model}</div>
      </div>
    </div>

    <div class="stage-right">
      <!-- Per-agent preview/apply controls -->
      <div class="stage-actions">
        <button
          class="btn-stage-preview"
          type="button"
          disabled={!canPreview}
          on:click={() => onPreview?.()}
          title="Run this agent only (preview — not committed)"
        >
          {previewRunning ? "Running…" : "Run This Agent"}
        </button>
        {#if preview}
          <button
            class="btn-stage-apply"
            type="button"
            disabled={!canApply}
            on:click={() => onApply?.()}
            title="Merge this agent's preview into the live canonical state"
          >
            Apply
          </button>
        {/if}
      </div>

      {#if status === "done"}
        <span class="stage-status s-done">✓ Complete</span>
      {:else if status === "running"}
        <span class="stage-status s-run">⟳ Running</span>
      {:else if status === "error"}
        <span class="stage-status s-err">✕ Error</span>
      {:else}
        <span class="stage-status s-wait">◍ Waiting</span>
      {/if}
    </div>
  </div>

  <!-- Preview banner — shown when a preview result exists -->
  {#if preview}
    <div class="preview-banner">
      <span class="chip-warn">Preview — not committed</span>
      <span class="preview-ts">
        {new Date(preview.created_at).toLocaleTimeString()}
      </span>
    </div>
    <div class="stage-body">
      <div class="stage-log preview-log">{previewOutputText}</div>
    </div>
  {/if}

  {#if passHistory.length > 0 || status === "running" || status === "error" || (status === "done" && summary)}
    <div class="stage-body">
      {#if passHistory.length > 0}
        <div class="chat-history" aria-label="Previous pass contributions">
          {#each passHistory as pass (pass.iteration)}
            <div class="chat-thread chat-thread-past">
              <div class="pass-chip">Pass {pass.iteration}</div>
              <div class="chat-row">
                <div class="chat-avatar chat-avatar-done" aria-hidden="true">{agentInitials}</div>
                <div class="chat-bubble chat-bubble-past">
                  <p class="chat-done-summary">{pass.headline}</p>
                  {#if pass.bullets.length > 0}
                    <ul class="chat-bullets">
                      {#each pass.bullets as item}
                        <li>{item}</li>
                      {/each}
                    </ul>
                  {/if}
                </div>
              </div>
            </div>
          {/each}
        </div>
      {/if}

      {#if status === "error" && errorMessage}
        <div class="stage-log stage-error-log">{errorMessage}</div>
      {:else if status === "running" && chatPresent}
        <div class="chat-thread chat-thread-live" aria-live="polite">
          {#if activePass > 0}
            <div class="pass-chip pass-chip-live">Pass {activePass} · In progress</div>
          {/if}
          <div class="chat-row">
            <div class="chat-avatar" aria-hidden="true">{agentInitials}</div>
            <div class="chat-bubble chat-bubble-live">
              <p class="chat-headline">
                {chatPresent.headline}<span class="stream-cursor" aria-hidden="true"></span>
              </p>
              {#if chatPresent.bullets.length > 0}
                <ul class="chat-bullets">
                  {#each chatPresent.bullets as item}
                    <li>{item}</li>
                  {/each}
                </ul>
              {:else if !chatPresent.hasContent}
                <div class="chat-typing" aria-hidden="true">
                  <span></span><span></span><span></span>
                </div>
              {/if}
            </div>
          </div>
        </div>
      {:else if status === "done" && summary && passHistory.length === 0}
        <div class="chat-thread chat-thread-done">
          <div class="pass-chip">Pass {activePass > 0 ? activePass : "—"}</div>
          <div class="chat-row">
            <div class="chat-avatar chat-avatar-done" aria-hidden="true">{agentInitials}</div>
            <div class="chat-bubble chat-bubble-done">
              <p class="chat-done-label">Contribution</p>
              <p class="chat-done-summary">{summary}</p>
              {#if summaryBullets.length > 0}
                <ul class="chat-bullets">
                  {#each summaryBullets as item}
                    <li>{item}</li>
                  {/each}
                </ul>
              {/if}
            </div>
          </div>
        </div>
      {:else if status === "waiting" && passHistory.length > 0 && pipelineRunning}
        <p class="up-next">Up next in this pass…</p>
      {/if}
    </div>
  {/if}
</div>

<style>
  .stage {
    padding: 16px 18px;
    background: var(--surface);
    border-radius: 14px;
    border: 1.5px solid var(--line-solid);
    box-shadow: 0 2px 8px rgba(35, 46, 82, 0.05);
    transition:
      border-color 0.25s,
      box-shadow 0.25s;
  }

  .stage-done {
    border-color: var(--ok-line);
    box-shadow: 0 2px 12px rgba(27, 159, 102, 0.08);
  }

  .stage-running {
    border-color: var(--accent-2);
    box-shadow: 0 2px 14px rgba(31, 122, 224, 0.12);
  }

  .stage-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .stage-right {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .stage-actions {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .btn-stage-preview {
    font-size: 0.75rem;
    font-weight: 600;
    padding: 5px 12px;
    border-radius: 999px;
    border: 1.5px solid var(--accent-2);
    background: var(--accent-2);
    color: var(--on-accent);
    cursor: pointer;
    white-space: nowrap;
    transition:
      background 0.15s,
      border-color 0.15s,
      box-shadow 0.15s;
  }

  .btn-stage-preview:hover:not(:disabled) {
    background: var(--accent-2-hover);
    border-color: var(--accent-2-hover);
    box-shadow: 0 2px 8px rgba(31, 122, 224, 0.35);
  }

  .btn-stage-preview:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .btn-stage-apply {
    font-size: 0.75rem;
    font-weight: 600;
    padding: 5px 10px;
    border-radius: 999px;
    border: 1px solid var(--ok);
    background: transparent;
    color: var(--ok);
    cursor: pointer;
    white-space: nowrap;
    transition:
      background 0.15s,
      color 0.15s;
  }

  .btn-stage-apply:hover:not(:disabled) {
    background: var(--ok);
    color: var(--on-accent);
  }

  .btn-stage-apply:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .preview-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
    margin-left: 40px;
  }

  .chip-warn {
    font-size: 0.7rem;
    font-weight: 700;
    padding: 3px 8px;
    border-radius: 999px;
    background: var(--warn-bg);
    color: var(--warn);
    border: 1px solid var(--warn-line);
    white-space: nowrap;
  }

  .preview-ts {
    font-size: 0.7rem;
    color: var(--ink-500);
  }

  .preview-log {
    margin-top: 6px;
    border-color: var(--warn-line) !important;
    background: var(--warn-bg) !important;
    opacity: 0.9;
  }

  .stage-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .stage-num {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    font-size: 0.8125rem;
    flex-shrink: 0;
    transition:
      background 0.3s,
      color 0.3s;
  }

  .stage-done .stage-num {
    background: var(--ok);
    color: var(--on-accent);
  }

  .stage-running .stage-num {
    background: var(--accent-2);
    color: var(--on-accent);
  }

  .stage-waiting .stage-num {
    background: var(--waiting-bg);
    color: var(--waiting-ink);
  }

  .stage-name {
    font-weight: 600;
    font-size: 0.875rem;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .stage-model {
    color: var(--ink-500);
    font-size: 0.75rem;
    margin-top: 2px;
  }

  .stage-status {
    font-size: 0.75rem;
    font-weight: 600;
    padding: 5px 10px;
    border-radius: 999px;
    white-space: nowrap;
    transition: all 0.3s;
  }

  .s-done {
    background: var(--ok-bg);
    color: var(--ok);
    border: 1px solid var(--ok-line);
  }

  .s-run {
    background: var(--accent-bg);
    color: var(--accent-2);
    border: 1px solid var(--accent-line);
  }

  .s-wait {
    background: var(--neutral-bg);
    color: var(--ink-500);
    border: 1px solid var(--line);
  }

  .stage-body {
    margin-top: 12px;
    margin-left: 40px;
  }

  .stage-log {
    font-family: "IBM Plex Mono", monospace;
    font-size: 0.75rem;
    color: var(--log-ink);
    background: var(--log-bg);
    border: 1px solid var(--log-line);
    border-radius: 9px;
    padding: 10px 12px;
    line-height: 1.65;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .stage-waiting {
    opacity: 0.45;
  }

  .dots {
    animation: blink 1.2s infinite;
  }

  .status-pulse {
    display: block;
    margin-top: 6px;
    font-size: 0.75rem;
    color: var(--ink-500);
    animation: blink 1.4s infinite;
  }

  .stream-log {
    position: relative;
    max-height: 280px;
    overflow-y: auto;
  }

  .chat-thread {
    margin-top: 4px;
  }

  .chat-history {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-bottom: 10px;
  }

  .chat-thread-past {
    opacity: 0.88;
  }

  .chat-thread-live {
    margin-top: 2px;
  }

  .pass-chip {
    display: inline-block;
    margin: 0 0 6px 46px;
    font-size: 0.68rem;
    font-weight: 700;
    letter-spacing: 0.05em;
    text-transform: uppercase;
    color: var(--ink-500);
    background: var(--neutral-bg);
    border: 1px solid var(--line-solid);
    border-radius: 999px;
    padding: 2px 8px;
  }

  .pass-chip-live {
    color: var(--accent-2);
    background: var(--accent-bg);
    border-color: var(--accent-line);
  }

  .chat-bubble-past {
    border-color: var(--ok-line);
    background: rgba(255, 255, 255, 0.78);
  }

  .up-next {
    margin: 8px 0 0 46px;
    font-size: 0.75rem;
    font-style: italic;
    color: var(--ink-500);
  }

  .chat-row {
    display: flex;
    gap: 12px;
    align-items: flex-start;
  }

  .chat-avatar {
    flex-shrink: 0;
    width: 34px;
    height: 34px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.02em;
    color: var(--on-accent);
    background: linear-gradient(145deg, var(--accent-2), var(--accent));
    box-shadow: 0 2px 8px rgba(31, 122, 224, 0.25);
  }

  .chat-avatar-done {
    background: linear-gradient(145deg, var(--ok), #2bc48a);
    box-shadow: 0 2px 8px rgba(27, 159, 102, 0.25);
  }

  .chat-bubble {
    flex: 1;
    min-width: 0;
    border-radius: 14px 14px 14px 4px;
    padding: 12px 14px;
    background: rgba(255, 255, 255, 0.92);
    border: 1px solid var(--line-solid);
    box-shadow: 0 2px 10px rgba(35, 46, 82, 0.06);
  }

  .chat-bubble-live {
    border-color: var(--accent-line);
    background: linear-gradient(180deg, #ffffff 0%, #f6faff 100%);
  }

  .chat-bubble-done {
    border-color: var(--ok-line);
    background: linear-gradient(180deg, #ffffff 0%, var(--ok-bg-soft) 100%);
  }

  .chat-headline {
    margin: 0;
    font-size: 0.875rem;
    line-height: 1.55;
    color: var(--ink-700);
    font-weight: 500;
  }

  .chat-bullets {
    margin: 8px 0 0;
    padding-left: 18px;
    list-style: disc;
  }

  .chat-bullets li {
    margin: 3px 0;
    font-size: 0.8125rem;
    line-height: 1.5;
    color: var(--ink-500);
  }

  .chat-typing {
    display: flex;
    gap: 5px;
    margin-top: 10px;
    align-items: center;
  }

  .chat-typing span {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--accent-2);
    opacity: 0.35;
    animation: typing-bounce 1.2s infinite ease-in-out;
  }

  .chat-typing span:nth-child(2) {
    animation-delay: 0.15s;
  }

  .chat-typing span:nth-child(3) {
    animation-delay: 0.3s;
  }

  .chat-done-label {
    margin: 0 0 4px;
    font-size: 0.7rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--ok-ink-strong);
  }

  .chat-done-summary {
    margin: 0;
    font-size: 0.875rem;
    line-height: 1.55;
    color: var(--ok-ink);
    font-weight: 500;
  }

  @keyframes typing-bounce {
    0%,
    80%,
    100% {
      transform: translateY(0);
      opacity: 0.35;
    }
    40% {
      transform: translateY(-4px);
      opacity: 1;
    }
  }

  .stage-error-log {
    color: #ffb4c4;
    border-color: rgba(206, 49, 88, 0.35);
    background: rgba(206, 49, 88, 0.12);
  }

  .s-err {
    color: var(--danger);
    background: rgba(206, 49, 88, 0.1);
    border: 1px solid rgba(206, 49, 88, 0.25);
    border-radius: 6px;
    padding: 2px 8px;
    font-size: 0.75rem;
    font-weight: 600;
  }

  .stage-error {
    border-left-color: var(--danger);
  }

  .stream-cursor {
    display: inline-block;
    width: 2px;
    height: 1em;
    background: var(--accent-2);
    vertical-align: text-bottom;
    margin-right: 3px;
    border-radius: 1px;
    animation: cursor-blink 0.9s steps(1) infinite;
  }

  @keyframes cursor-blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0; }
  }

  @keyframes blink {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.25;
    }
  }
</style>
