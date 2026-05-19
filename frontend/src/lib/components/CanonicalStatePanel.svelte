<script lang="ts">
  import type { CanonicalState } from "$lib/types";

  /** Canonical state to display. Null renders an empty / loading placeholder. */
  export let state: CanonicalState | null = null;

  let expandedPlan = false;
  let expandedAssumptions = false;
  let expandedQuestions = false;

  $: planSteps = state?.execution_plan ?? [];
  $: assumptions = state?.assumptions ?? [];
  $: openQuestions = state?.open_questions ?? [];
</script>

{#if !state}
  <div class="card empty-state">
    <p style="color:var(--ink-500);font-size:0.875rem;margin:0;">
      No state yet — run the first iteration to populate.
    </p>
  </div>
{:else}
  <div class="state-panel">
    <!-- Idea -->
    {#if state.idea?.text}
      <div class="card mini-card">
        <div class="mini-label">Idea</div>
        <div class="mini-body">{state.idea.text}</div>
      </div>
    {/if}

    <!-- Architecture -->
    {#if state.architecture?.overview}
      <div class="card mini-card">
        <div class="mini-label">Architecture</div>
        <div class="mini-body">{state.architecture.overview}</div>
        {#if state.architecture.components?.length}
          <ul class="mini-list">
            {#each state.architecture.components.slice(0, 4) as comp}
              <li>{comp}</li>
            {/each}
            {#if state.architecture.components.length > 4}
              <li style="color:var(--ink-300);">
                +{state.architecture.components.length - 4} more
              </li>
            {/if}
          </ul>
        {/if}
      </div>
    {/if}

    <!-- Execution Plan -->
    {#if planSteps.length > 0}
      <div class="card mini-card">
        <button
          class="mini-accordion-btn"
          on:click={() => (expandedPlan = !expandedPlan)}
          type="button"
        >
          <div class="mini-label" style="pointer-events:none;">
            Execution Plan
          </div>
          <span class="mini-count">{planSteps.length} steps</span>
          <span class="mini-chevron">{expandedPlan ? "▲" : "▼"}</span>
        </button>
        {#if expandedPlan}
          <ol class="plan-list">
            {#each planSteps as step, i (step.id ?? i)}
              <li>
                <strong>{step.title}</strong>
                {#if step.description}
                  <span style="color:var(--ink-500);">— {step.description}</span
                  >
                {/if}
              </li>
            {/each}
          </ol>
        {/if}
      </div>
    {/if}

    <!-- Assumptions -->
    {#if assumptions.length > 0}
      <div class="card mini-card">
        <button
          class="mini-accordion-btn"
          on:click={() => (expandedAssumptions = !expandedAssumptions)}
          type="button"
        >
          <div class="mini-label" style="pointer-events:none;">Assumptions</div>
          <span class="mini-count">{assumptions.length}</span>
          <span class="mini-chevron">{expandedAssumptions ? "▲" : "▼"}</span>
        </button>
        {#if expandedAssumptions}
          <ul class="mini-list">
            {#each assumptions as a}
              <li>{a}</li>
            {/each}
          </ul>
        {/if}
      </div>
    {/if}

    <!-- Open Questions -->
    {#if openQuestions.length > 0}
      <div class="card mini-card">
        <button
          class="mini-accordion-btn"
          on:click={() => (expandedQuestions = !expandedQuestions)}
          type="button"
        >
          <div class="mini-label" style="pointer-events:none;">
            Open Questions
          </div>
          <span class="mini-count">{openQuestions.length}</span>
          <span class="mini-chevron">{expandedQuestions ? "▲" : "▼"}</span>
        </button>
        {#if expandedQuestions}
          <ul class="mini-list">
            {#each openQuestions as q}
              <li>{q}</li>
            {/each}
          </ul>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
  .state-panel {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .empty-state {
    padding: 20px;
  }

  .mini-card {
    padding: 14px 16px;
  }

  .mini-label {
    font-weight: 600;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--ink-500);
    margin-bottom: 6px;
  }

  .mini-body {
    font-size: 0.875rem;
    color: var(--ink-700);
    line-height: 1.5;
  }

  .mini-list {
    margin: 6px 0 0 16px;
    padding: 0;
    font-size: 0.8125rem;
    color: var(--ink-700);
    line-height: 1.6;
  }

  .mini-accordion-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    background: none;
    border: none;
    padding: 0;
    cursor: pointer;
    font-family: inherit;
  }

  .mini-count {
    font-size: 0.72rem;
    background: #e7eef9;
    color: var(--ink-700);
    border-radius: 20px;
    padding: 1px 7px;
    font-weight: 600;
    margin-left: auto;
  }

  .mini-chevron {
    font-size: 0.625rem;
    color: var(--ink-300);
  }

  .plan-list {
    margin: 8px 0 0 16px;
    padding: 0;
    font-size: 0.8125rem;
    color: var(--ink-700);
    line-height: 1.6;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
</style>
