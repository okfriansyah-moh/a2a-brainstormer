<!-- @deprecated — Inlined into session/[id]/+page.svelte. Kept for backwards-compatibility only. -->
<script lang="ts">
  import type { AgentMeta } from "$lib/types";

  /** One snapshot per completed pipeline pass. */
  export let iterations: { iteration: number; agents: AgentMeta[] }[] = [];

  const roleBadgeClass: Record<string, string> = {
    build: "bg-blue-100 text-blue-700",
    review: "bg-yellow-100 text-yellow-700",
    refine: "bg-green-100 text-green-700",
    devils_advocate: "bg-red-100 text-red-700",
  };
</script>

<div class="rounded-lg border border-gray-200 bg-white p-4">
  <h2 class="mb-3 text-sm font-semibold text-gray-900">Iteration Timeline</h2>

  {#if iterations.length === 0}
    <p class="text-sm text-gray-400 italic">No iterations yet.</p>
  {:else}
    <div class="overflow-x-auto">
      <div class="flex min-w-max gap-px">
        {#each iterations as pass}
          <div class="flex flex-col items-center gap-1.5 px-3 first:pl-0">
            <!-- Iteration number badge -->
            <div
              class="flex h-6 w-6 items-center justify-center rounded-full bg-blue-600 text-xs font-bold text-white"
            >
              {pass.iteration}
            </div>

            <!-- Connector line (not for last) -->
            {#if pass !== iterations[iterations.length - 1]}
              <div class="absolute"></div>
            {/if}

            <!-- Agent roles in this pass -->
            <div class="flex flex-col gap-1">
              {#each pass.agents as agent}
                <div
                  class="rounded border border-gray-100 bg-gray-50 px-2 py-1"
                >
                  <p
                    class="text-xs font-medium text-gray-700 truncate max-w-28"
                  >
                    {agent.name}
                  </p>
                  <span
                    class="mt-0.5 inline-block rounded-full px-1.5 py-0.5 text-xs {roleBadgeClass[
                      agent.role
                    ] ?? 'bg-gray-100 text-gray-600'}"
                  >
                    {agent.role}
                  </span>
                </div>
              {/each}
            </div>
          </div>

          <!-- Horizontal connector between passes -->
          {#if pass !== iterations[iterations.length - 1]}
            <div class="flex items-start pt-3">
              <div class="h-px w-6 bg-gray-300 mt-3"></div>
            </div>
          {/if}
        {/each}
      </div>
    </div>
  {/if}
</div>
