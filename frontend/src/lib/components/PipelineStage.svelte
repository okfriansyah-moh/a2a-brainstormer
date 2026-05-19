<script lang="ts">
  import type { SessionAgent } from "$lib/types";

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

  $: roleCssClass = agent.role.replace(/_/g, "-").toLowerCase();
  $: badgeLabel = agent.role.replace(/_/g, " ").toUpperCase();

  /** Normalise output text: collapse excess whitespace, limit length for display. */
  $: displayOutput = output ? output.slice(0, 2000) : "";
</script>

<div class="stage stage-{status}">
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

    {#if status === "done"}
      <span class="stage-status s-done">✓ Complete</span>
    {:else if status === "running"}
      <span class="stage-status s-run">⟳ Running</span>
    {:else}
      <span class="stage-status s-wait">◍ Waiting</span>
    {/if}
  </div>

  {#if status !== "waiting" && (displayOutput || summary)}
    <div class="stage-body">
      {#if displayOutput}
        <div class="stage-log">{displayOutput}</div>
      {/if}
      {#if status === "done" && summary}
        <div class="stage-summary">
          <strong>Contribution:</strong>
          {summary}
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
  }

  .stage-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
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
    color: #fff;
  }

  .stage-running .stage-num {
    background: var(--accent-2);
    color: #fff;
  }

  .stage-waiting .stage-num {
    background: #e7eefc;
    color: #2b4f8f;
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
    background: #e6f7ef;
    color: var(--ok);
    border: 1px solid #b8e8d0;
  }

  .s-run {
    background: #edf6ff;
    color: var(--accent-2);
    border: 1px solid #b8d8ff;
  }

  .s-wait {
    background: #f3f5fa;
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
    color: #415070;
    background: #f9fbff;
    border: 1px solid #dae2f2;
    border-radius: 9px;
    padding: 10px 12px;
    line-height: 1.65;
    white-space: pre-wrap;
    word-break: break-word;
  }

  .stage-summary {
    margin-top: 8px;
    background: #f0f9f4;
    border: 1px solid #b8e8d0;
    border-radius: 9px;
    padding: 9px 12px;
    font-size: 0.8125rem;
    color: #1a7a50;
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
