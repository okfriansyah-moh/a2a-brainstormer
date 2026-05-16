<script lang="ts">
  export let loading: boolean = false;
  export let sessionStarted: boolean = false;
  export let converged: boolean = false;
  export let onNextIteration: () => void;
  export let onApprove: () => void;
  export let onInjectFeedback: (feedback: string) => void;

  let feedbackText = "";
  let showFeedback = false;

  function handleInjectFeedback() {
    if (feedbackText.trim()) {
      onInjectFeedback(feedbackText.trim());
      feedbackText = "";
      showFeedback = false;
    }
  }

  $: iterateDisabled = loading || !sessionStarted || converged;
  $: approveDisabled = loading || !sessionStarted;
</script>

<div class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
  <div class="flex flex-wrap items-center gap-3">
    <button
      class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
      disabled={iterateDisabled}
      on:click={onNextIteration}
    >
      {#if loading}
        <span class="flex items-center gap-1.5">
          <svg
            class="h-4 w-4 animate-spin"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
            ></path>
          </svg>
          Running…
        </span>
      {:else if converged}
        Converged ✓
      {:else}
        Next Iteration
      {/if}
    </button>

    <button
      class="rounded-md bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:cursor-not-allowed disabled:opacity-50"
      disabled={approveDisabled}
      on:click={onApprove}
    >
      Approve &amp; Finalize
    </button>

    <button
      class="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
      disabled={approveDisabled}
      on:click={() => (showFeedback = !showFeedback)}
    >
      Inject Feedback
    </button>

    {#if converged}
      <span class="text-xs font-medium text-green-700">
        ✓ Convergence reached — agents agree. Finalize or iterate further.
      </span>
    {/if}
  </div>

  {#if showFeedback}
    <div class="mt-3 flex gap-2">
      <textarea
        class="flex-1 resize-none rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
        rows={2}
        placeholder="Enter feedback to inject into the next iteration…"
        bind:value={feedbackText}
      ></textarea>
      <div class="flex flex-col gap-1">
        <button
          class="rounded-md bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          disabled={!feedbackText.trim()}
          on:click={handleInjectFeedback}
        >
          Inject
        </button>
        <button
          class="rounded-md border border-gray-300 px-3 py-1.5 text-xs text-gray-600 hover:bg-gray-50"
          on:click={() => (showFeedback = false)}
        >
          Cancel
        </button>
      </div>
    </div>
  {/if}
</div>
