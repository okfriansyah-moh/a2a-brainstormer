<!-- @deprecated — Use PipelineStage instead. Kept for backwards-compatibility only. -->
<script lang="ts">
  import type { SessionAgent, CanonicalState } from "$lib/types";

  export let agent: SessionAgent;
  export let previousOutput: CanonicalState | undefined = undefined;

  const roleBadgeClass: Record<string, string> = {
    build: "bg-blue-100 text-blue-800",
    review: "bg-yellow-100 text-yellow-800",
    refine: "bg-green-100 text-green-800",
    devils_advocate: "bg-red-100 text-red-800",
  };

  /** Compute simple line-level diff between two JSON strings. */
  function computeDiff(
    prev: CanonicalState | undefined,
    curr: CanonicalState | undefined,
  ): { type: "unchanged" | "added" | "removed"; text: string }[] {
    const prevLines = prev ? JSON.stringify(prev, null, 2).split("\n") : [];
    const currLines = curr ? JSON.stringify(curr, null, 2).split("\n") : [];

    const result: { type: "unchanged" | "added" | "removed"; text: string }[] =
      [];
    const prevSet = new Set(prevLines);
    const currSet = new Set(currLines);

    for (const line of currLines) {
      if (!prevSet.has(line)) {
        result.push({ type: "added", text: line });
      } else {
        result.push({ type: "unchanged", text: line });
      }
    }
    for (const line of prevLines) {
      if (!currSet.has(line)) {
        result.push({ type: "removed", text: line });
      }
    }
    return result;
  }

  $: hasDiff = previousOutput !== undefined && agent.output !== undefined;
  $: diffLines = hasDiff ? computeDiff(previousOutput, agent.output) : [];
  $: badgeClass = roleBadgeClass[agent.role] ?? "bg-gray-100 text-gray-800";
</script>

<div
  class="flex h-full min-w-72 flex-col rounded-lg border border-gray-200 bg-white shadow-sm"
>
  <!-- Header -->
  <div class="border-b border-gray-200 p-4">
    <div class="flex items-start justify-between gap-2">
      <h3 class="truncate text-sm font-semibold text-gray-900">{agent.name}</h3>
      <span
        class="shrink-0 rounded-full px-2 py-0.5 text-xs font-medium {badgeClass}"
      >
        {agent.role}
      </span>
    </div>
    <p class="mt-1 text-xs text-gray-500">{agent.provider} / {agent.model}</p>
    {#if agent.skills.length > 0}
      <div class="mt-2 flex flex-wrap gap-1">
        {#each agent.skills as skill}
          <span class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600"
            >{skill}</span
          >
        {/each}
      </div>
    {/if}
  </div>

  <!-- Output -->
  <div class="flex-1 overflow-auto p-4">
    {#if !agent.output}
      <p class="text-sm text-gray-400 italic">
        No output yet. Run an iteration to see results.
      </p>
    {:else if hasDiff}
      <!-- Diff view: added lines highlighted green, removed red -->
      <pre
        class="overflow-x-auto text-xs leading-relaxed">{#each diffLines as line}<span
            class={line.type === "added"
              ? "block bg-green-50 text-green-800"
              : line.type === "removed"
                ? "block bg-red-50 text-red-700 line-through opacity-60"
                : "block text-gray-700"}>{line.text}</span
          >{/each}</pre>
    {:else}
      <pre
        class="overflow-x-auto text-xs leading-relaxed text-gray-700">{JSON.stringify(
          agent.output,
          null,
          2,
        )}</pre>
    {/if}
  </div>
</div>
