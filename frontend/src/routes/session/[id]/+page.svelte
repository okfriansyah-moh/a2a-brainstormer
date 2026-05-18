<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import { sessionStore } from "$lib/stores/sessionStore";
  import AgentPanel from "$lib/components/AgentPanel.svelte";
  import ControlPanel from "$lib/components/ControlPanel.svelte";
  import StateView from "$lib/components/StateView.svelte";
  import Timeline from "$lib/components/Timeline.svelte";
  import { getSession, iterate, finalizeSession } from "$lib/services/api";
  import type { AgentMeta, CanonicalState } from "$lib/types";

  /** History of (iteration, agents) snapshots for the Timeline component. */
  let iterationHistory: { iteration: number; agents: AgentMeta[] }[] = [];

  /** Previous canonical state — used for diff highlights in AgentPanel. */
  let previousState: CanonicalState | undefined = undefined;

  /** Convergence flag — set to true when the backend signals convergence. */
  let converged = false;

  let loadError = "";
  let actionError = "";

  $: sessionId = $page.params.id;
  $: sessionStarted = !!$sessionStore.state;

  onMount(async () => {
    if (!sessionId) return;
    sessionStore.setLoading(true);
    loadError = "";
    try {
      const session = await getSession(sessionId);
      sessionStore.setSession(session.id, session.idea);
      if (session.current_state) {
        sessionStore.updateState(session.current_state);
        const meta = session.current_state.meta;
        if (meta?.iteration && meta?.agents) {
          iterationHistory = [
            { iteration: meta.iteration, agents: meta.agents },
          ];
        }
      }
      // Populate agents from state metadata if present
      if (session.current_state?.meta?.agents) {
        const agentSlots = session.current_state.meta.agents.map((a) => ({
          id: a.agent_id,
          name: a.name,
          role: a.role,
          provider: a.provider,
          model: a.model,
          skills: a.skills,
        }));
        sessionStore.setAgents(agentSlots);
      }
    } catch (err) {
      loadError =
        err instanceof Error ? err.message : "Failed to load session.";
    } finally {
      sessionStore.setLoading(false);
    }
  });

  async function handleNextIteration() {
    if ($sessionStore.loading || !sessionId) return;
    sessionStore.setLoading(true);
    actionError = "";
    previousState = $sessionStore.state ?? undefined;
    try {
      const result = await iterate(sessionId);
      sessionStore.updateState(result.state);
      converged = result.converged;

      const meta = result.state.meta;
      if (meta?.iteration && meta?.agents) {
        iterationHistory = [
          ...iterationHistory,
          { iteration: meta.iteration, agents: meta.agents },
        ];
      }
    } catch (err) {
      actionError = err instanceof Error ? err.message : "Iteration failed.";
    } finally {
      sessionStore.setLoading(false);
    }
  }

  async function handleApprove() {
    if ($sessionStore.loading || !sessionId) return;
    sessionStore.setLoading(true);
    actionError = "";
    try {
      await finalizeSession(sessionId);
      await goto("/");
    } catch (err) {
      actionError =
        err instanceof Error ? err.message : "Failed to finalize session.";
      sessionStore.setLoading(false);
    }
  }

  function handleInjectFeedback(feedback: string) {
    // Feedback injection is handled by the backend via the idea field on the
    // next iterate call. For now we surface the field for UX and note that
    // full support is wired in Task 15 integration.
    actionError = "";
    console.info("Feedback queued for next iteration:", feedback);
  }
</script>

<div class="flex h-screen flex-col overflow-hidden bg-gray-50">
  <!-- Top bar -->
  <header
    class="flex items-center justify-between border-b border-gray-200 bg-white px-6 py-3"
  >
    <div class="flex items-center gap-3">
      <a href="/" class="text-sm text-gray-500 hover:text-gray-700">← Home</a>
      <span class="text-gray-300">|</span>
      <h1 class="text-sm font-semibold text-gray-900">Session Workspace</h1>
      {#if $sessionStore.idea}
        <span
          class="max-w-sm truncate text-xs text-gray-500"
          title={$sessionStore.idea}
        >
          {$sessionStore.idea}
        </span>
      {/if}
    </div>
    <div class="flex items-center gap-2 text-xs text-gray-500">
      {#if $sessionStore.state?.meta?.iteration}
        Iteration {$sessionStore.state.meta.iteration}
      {/if}
      {#if converged}
        <span
          class="rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700"
        >
          Converged
        </span>
      {/if}
    </div>
  </header>

  <!-- Load error -->
  {#if loadError}
    <div
      class="mx-6 mt-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
    >
      {loadError}
    </div>
  {/if}

  <!-- Action error -->
  {#if actionError}
    <div
      class="mx-6 mt-4 rounded-md border border-orange-200 bg-orange-50 px-4 py-3 text-sm text-orange-700"
    >
      {actionError}
    </div>
  {/if}

  <!-- Main workspace -->
  <div class="flex flex-1 flex-col overflow-hidden">
    <!-- Agent panels — horizontal scrollable row -->
    {#if $sessionStore.agents.length > 0}
      <div
        class="flex gap-4 overflow-x-auto border-b border-gray-200 bg-white p-4"
      >
        {#each $sessionStore.agents as agent}
          <AgentPanel {agent} previousOutput={previousState} />
        {/each}
      </div>
    {:else if !$sessionStore.loading}
      <div class="border-b border-gray-200 bg-white px-6 py-4">
        <p class="text-sm text-gray-400 italic">
          No agents in this session. Run the first iteration to populate agent
          data.
        </p>
      </div>
    {/if}

    <!-- Scrollable content area -->
    <div class="flex-1 overflow-y-auto p-6 space-y-4">
      <!-- Control panel -->
      <ControlPanel
        loading={$sessionStore.loading}
        {sessionStarted}
        {converged}
        onNextIteration={handleNextIteration}
        onApprove={handleApprove}
        onInjectFeedback={handleInjectFeedback}
      />

      <!-- Two-column layout: state view + timeline -->
      <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div class="lg:col-span-2">
          <StateView state={$sessionStore.state} />
        </div>
        <div class="lg:col-span-1">
          <Timeline iterations={iterationHistory} />
        </div>
      </div>
    </div>
  </div>
</div>
