<script lang="ts">
  import type { Risk } from "$lib/types";

  /** List of risks from canonical state. */
  export let risks: Risk[] = [];

  $: criticalCount = risks.filter((r) => r.severity === "critical").length;
  $: highCount = risks.filter((r) => r.severity === "high").length;
  $: mediumCount = risks.filter((r) => r.severity === "medium").length;
  $: lowCount = risks.filter((r) => r.severity === "low").length;

  function chipClass(sev: Risk["severity"]): string {
    switch (sev) {
      case "critical":
        return "chip-danger";
      case "high":
        return "chip-warn";
      case "medium":
        return "chip-ok";
      default:
        return "chip-live";
    }
  }
</script>

<div class="risk-board">
  {#if risks.length === 0}
    <div class="empty">
      <span class="empty-icon">🛡️</span>
      <p>No risks identified</p>
    </div>
  {:else}
    <!-- Severity summary row -->
    <div class="summary-row">
      {#if criticalCount > 0}
        <span class="chip-danger sev-chip">{criticalCount} Critical</span>
      {/if}
      {#if highCount > 0}
        <span class="chip-warn sev-chip">{highCount} High</span>
      {/if}
      {#if mediumCount > 0}
        <span class="chip-ok sev-chip">{mediumCount} Medium</span>
      {/if}
      {#if lowCount > 0}
        <span class="chip-live sev-chip">{lowCount} Low</span>
      {/if}
    </div>

    <!-- Risk list -->
    <ul class="risk-list">
      {#each risks as risk (risk.id ?? risk.title)}
        <li class="risk-item" class:resolved={risk.resolved}>
          <div class="risk-header">
            <span class="risk-title">{risk.title}</span>
            <span class="{chipClass(risk.severity)} sev-badge">
              {risk.severity}
            </span>
          </div>
          {#if risk.description}
            <div class="risk-desc">{risk.description}</div>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .risk-board {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 28px 16px;
    gap: 8px;
    color: var(--ink-300);
    font-size: 0.875rem;
  }

  .empty-icon {
    font-size: 1.75rem;
  }

  .empty p {
    margin: 0;
  }

  .summary-row {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }

  .sev-chip {
    font-size: 0.75rem;
    font-weight: 700;
    border-radius: 999px;
    padding: 3px 10px;
    display: inline-flex;
    align-items: center;
  }

  .sev-badge {
    font-size: 0.6875rem;
    font-weight: 700;
    border-radius: 999px;
    padding: 2px 8px;
    display: inline-flex;
    align-items: center;
    text-transform: capitalize;
    flex-shrink: 0;
  }

  .risk-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .risk-item {
    background: rgba(255, 255, 255, 0.55);
    border: 1px solid rgba(168, 174, 199, 0.2);
    border-radius: 10px;
    padding: 10px 12px;
  }

  .risk-item.resolved {
    opacity: 0.45;
  }

  .risk-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  .risk-title {
    font-weight: 600;
    font-size: 0.8125rem;
    color: var(--ink-900);
    flex: 1;
    min-width: 0;
  }

  .risk-desc {
    font-size: 0.75rem;
    color: var(--ink-500);
    margin-top: 4px;
    line-height: 1.5;
  }
</style>
