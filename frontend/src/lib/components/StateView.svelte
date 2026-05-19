<!-- @deprecated — Use CanonicalStatePanel instead. Kept for backwards-compatibility only. -->
<script lang="ts">
  import type { CanonicalState } from "$lib/types";

  export let state: CanonicalState | null;

  const severityClass: Record<string, string> = {
    low: "bg-green-100 text-green-800",
    medium: "bg-yellow-100 text-yellow-800",
    high: "bg-orange-100 text-orange-800",
    critical: "bg-red-100 text-red-800",
  };

  let expandedPlan = false;
</script>

{#if !state}
  <div class="rounded-lg border border-gray-200 bg-white p-6">
    <p class="text-sm text-gray-400 italic">
      No state yet. Run the first iteration to see results.
    </p>
  </div>
{:else}
  <div class="space-y-4">
    <!-- Idea -->
    {#if state.idea?.text}
      <div class="rounded-lg border border-gray-200 bg-white p-4">
        <h2 class="mb-1 text-sm font-semibold text-gray-900">Idea</h2>
        <p class="text-sm text-gray-700">{state.idea.text}</p>
      </div>
    {/if}

    <!-- Architecture -->
    {#if state.architecture?.overview}
      <div class="rounded-lg border border-gray-200 bg-white p-4">
        <h2 class="mb-2 text-sm font-semibold text-gray-900">Architecture</h2>
        <p class="mb-3 text-sm text-gray-700">{state.architecture.overview}</p>
        {#if state.architecture.components && state.architecture.components.length > 0}
          <ul class="space-y-1">
            {#each state.architecture.components as comp}
              <li class="flex items-start gap-2 text-sm">
                <span
                  class="mt-0.5 h-1.5 w-1.5 shrink-0 rounded-full bg-blue-500"
                ></span>
                <span class="text-gray-700">{comp}</span>
              </li>
            {/each}
          </ul>
        {/if}
        {#if state.architecture.decisions && state.architecture.decisions.length > 0}
          <h3
            class="mt-3 text-xs font-semibold text-gray-600 uppercase tracking-wide"
          >
            Decisions
          </h3>
          <ul class="mt-1 space-y-1">
            {#each state.architecture.decisions as decision}
              <li class="flex items-start gap-2 text-sm text-gray-700">
                <span class="mt-1 h-1 w-1 shrink-0 rounded-full bg-gray-400"
                ></span>
                {decision}
              </li>
            {/each}
          </ul>
        {/if}
      </div>
    {/if}

    <!-- Execution Plan -->
    {#if state.execution_plan && state.execution_plan.length > 0}
      <div class="rounded-lg border border-gray-200 bg-white">
        <button
          class="flex w-full items-center justify-between p-4 text-left"
          on:click={() => (expandedPlan = !expandedPlan)}
        >
          <h2 class="text-sm font-semibold text-gray-900">
            Execution Plan
            <span class="ml-1 text-xs font-normal text-gray-500">
              ({state.execution_plan.length} steps)
            </span>
          </h2>
          <svg
            class="h-4 w-4 text-gray-500 transition-transform {expandedPlan
              ? 'rotate-180'
              : ''}"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M19 9l-7 7-7-7"
            ></path>
          </svg>
        </button>
        {#if expandedPlan}
          <div class="border-t border-gray-200 px-4 pb-4">
            <ol class="mt-3 space-y-2">
              {#each state.execution_plan as step, i}
                <li class="flex items-start gap-3 text-sm">
                  <span
                    class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-blue-100 text-xs font-medium text-blue-700"
                  >
                    {i + 1}
                  </span>
                  <div>
                    <p class="font-medium text-gray-800">{step.title}</p>
                    {#if step.description}
                      <p class="text-xs text-gray-500">{step.description}</p>
                    {/if}
                  </div>
                </li>
              {/each}
            </ol>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Risks -->
    {#if state.risks && state.risks.length > 0}
      <div class="rounded-lg border border-gray-200 bg-white p-4">
        <h2 class="mb-3 text-sm font-semibold text-gray-900">
          Risks <span class="text-xs font-normal text-gray-500"
            >({state.risks.length})</span
          >
        </h2>
        <ul class="space-y-2">
          {#each state.risks as risk}
            <li class="flex items-start gap-2 text-sm">
              <span
                class="mt-0.5 shrink-0 rounded px-1.5 py-0.5 text-xs font-medium {severityClass[
                  risk.severity
                ] ?? 'bg-gray-100 text-gray-700'}"
              >
                {risk.severity}
              </span>
              <span class="text-gray-700">{risk.description}</span>
            </li>
          {/each}
        </ul>
      </div>
    {/if}

    <!-- Assumptions -->
    {#if state.assumptions && state.assumptions.length > 0}
      <div class="rounded-lg border border-gray-200 bg-white p-4">
        <h2 class="mb-2 text-sm font-semibold text-gray-900">
          Assumptions <span class="text-xs font-normal text-gray-500"
            >({state.assumptions.length})</span
          >
        </h2>
        <ul class="space-y-1">
          {#each state.assumptions as assumption}
            <li class="flex items-start gap-2 text-sm text-gray-700">
              <span class="mt-1.5 h-1 w-1 shrink-0 rounded-full bg-gray-400"
              ></span>
              {assumption}
            </li>
          {/each}
        </ul>
      </div>
    {/if}

    <!-- Open Questions -->
    {#if state.open_questions && state.open_questions.length > 0}
      <div class="rounded-lg border border-gray-200 bg-white p-4">
        <h2 class="mb-2 text-sm font-semibold text-gray-900">
          Open Questions <span class="text-xs font-normal text-gray-500"
            >({state.open_questions.length})</span
          >
        </h2>
        <ul class="space-y-1">
          {#each state.open_questions as q}
            <li class="flex items-start gap-2 text-sm text-gray-700">
              <span class="mt-0.5 shrink-0 text-gray-400">?</span>
              {q}
            </li>
          {/each}
        </ul>
      </div>
    {/if}

    <!-- Metrics -->
    {#if state.metrics}
      <div class="rounded-lg border border-gray-200 bg-white p-4">
        <h2 class="mb-2 text-sm font-semibold text-gray-900">Convergence</h2>
        <div class="flex items-center gap-3">
          <div class="flex-1 overflow-hidden rounded-full bg-gray-200">
            <div
              class="h-2 rounded-full bg-blue-500 transition-all duration-500"
              style="width: {Math.round(
                (state.metrics.confidence ?? 0) * 100,
              )}%"
            ></div>
          </div>
          <span class="text-sm font-medium text-gray-700">
            {Math.round((state.metrics.confidence ?? 0) * 100)}%
          </span>
        </div>
      </div>
    {/if}
  </div>
{/if}
