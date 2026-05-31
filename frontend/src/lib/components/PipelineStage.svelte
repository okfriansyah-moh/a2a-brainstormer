<script lang="ts">
  import type { PreviewResult, SessionAgent } from "$lib/types";

  /** The agent represented by this stage. */
  export let agent: SessionAgent;

  /** Stage position number (1-based). */
  export let position: number;

  /** Execution state of this stage. */
  export let status: "done" | "running" | "waiting" = "waiting";

  /** Mono log text emitted by this agent (shown when running or done). */
  export let output: string = "";

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

  /** Normalise output text: collapse excess whitespace, limit length for display. */
  $: displayOutput = output ? output.slice(0, 2000) : "";

  $: previewOutputText = preview
    ? JSON.stringify(preview.output, null, 2).slice(0, 1500)
    : "";

  $: canPreview = !pipelineRunning && !previewRunning;
  $: canApply = !pipelineRunning && !previewRunning && !!preview;
</script>

<div class="stage stage-{status}" role="region" aria-label={agent.name}>
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

  {#if status !== "waiting" && (displayOutput || summary)}
    <div class="stage-body">
      {#if displayOutput}
        <div class="stage-log">{displayOutput}</div>
      {/if}
      {#if status === "done" && summary}
        <div class="stage-summary">
          <div class="stage-summary-head">
            <strong>Contribution</strong>
            <span class="stage-summary-text">{summary}</span>
          </div>
          {#if summaryBullets.length > 0}
            <ul class="stage-summary-list">
              {#each summaryBullets as item}
                <li>{item}</li>
              {/each}
            </ul>
          {/if}
        </div>
      {:else if status === "running" && !displayOutput}
        <div class="stage-log">
          <span class="dots">Processing...</span>
        </div>
      {/if}
    </div>
  {:else if status === "running" && !displayOutput}
    <div class="stage-body">
      <div class="stage-log"><span class="dots">Processing...</span></div>
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

  .stage-summary {
    margin-top: 8px;
    background: var(--ok-bg-soft);
    border: 1px solid var(--ok-line);
    border-radius: 9px;
    padding: 10px 14px;
    font-size: 0.8125rem;
    color: var(--ok-ink);
    word-break: break-word;
    line-height: 1.55;
  }

  .stage-summary-head {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: baseline;
  }

  .stage-summary-head strong {
    color: var(--ok-ink-strong);
    font-weight: 600;
  }

  .stage-summary-head strong::after {
    content: ":";
  }

  .stage-summary-text {
    flex: 1;
    min-width: 0;
  }

  .stage-summary-list {
    margin: 6px 0 0;
    padding-left: 20px;
    list-style: disc;
  }

  .stage-summary-list li {
    margin: 2px 0;
    line-height: 1.5;
  }

  .stage-waiting {
    opacity: 0.45;
  }

  .dots {
    animation: blink 1.2s infinite;
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
