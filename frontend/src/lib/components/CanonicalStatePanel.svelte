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

  /**
   * Split text that embeds a numbered list like
   * "Preamble: 1. Foo bar 2. Baz qux 3. …" into
   * { intro: "Preamble:", items: ["Foo bar", "Baz qux", …] }.
   * Returns null when no embedded numbering is detected.
   */
  function parseIdeaItems(
    text: string,
  ): { intro: string; items: string[] } | null {
    const idx = text.search(/\s1\.\s/);
    if (idx === -1) return null;
    const intro = text.slice(0, idx).trim();
    const itemsText = text.slice(idx);
    const items = itemsText
      .split(/\s+(?=\d+\.\s)/)
      .map((s) => s.replace(/^\d+\.\s+/, "").trim())
      .filter(Boolean);
    return items.length > 1 ? { intro, items } : null;
  }

  $: ideaParsed = state?.idea?.text ? parseIdeaItems(state.idea.text) : null;
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
        {#if ideaParsed}
          {#if ideaParsed.intro}
            <p class="mini-body" style="margin:0 0 8px;">{ideaParsed.intro}</p>
          {/if}
          <ol class="numbered-list">
            {#each ideaParsed.items as item, i}
              <li>
                <span class="item-label">{i + 1}.</span>
                {item}
              </li>
            {/each}
          </ol>
        {:else}
          <div class="mini-body">{state.idea.text}</div>
        {/if}
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
          <ol class="numbered-list">
            {#each assumptions as a, i}
              <li>
                <span class="item-label">Assumption {i + 1}:</span>
                {a}
              </li>
            {/each}
          </ol>
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
          <ol class="numbered-list">
            {#each openQuestions as q, i}
              <li>
                <span class="item-label">Open question {i + 1}:</span>
                {q}
              </li>
            {/each}
          </ol>
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

  .numbered-list {
    list-style: none;
    margin: 8px 0 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .numbered-list li {
    font-size: 0.8125rem;
    color: var(--ink-700);
    line-height: 1.6;
    padding-left: 2px;
  }

  .item-label {
    font-weight: 700;
    color: var(--ink-900);
    margin-right: 4px;
  }
</style>
