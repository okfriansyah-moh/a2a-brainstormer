<script lang="ts">
  import { onMount } from "svelte";
  import { agentRegistryStore } from "$lib/stores/agentRegistryStore";
  import SkillManager from "$lib/components/SkillManager.svelte";
  import { getAgents, getSkills } from "$lib/services/api";

  let error = "";

  onMount(async () => {
    agentRegistryStore.setLoading(true);
    error = "";
    try {
      const [agents, skills] = await Promise.all([getAgents(), getSkills()]);
      agentRegistryStore.setAgents(agents);
      agentRegistryStore.setSkills(skills);
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to load data.";
    } finally {
      agentRegistryStore.setLoading(false);
    }
  });
</script>

<div class="min-h-screen bg-gray-50">
  <!-- Header -->
  <header class="border-b border-gray-200 bg-white px-8 py-4">
    <div class="flex items-center justify-between">
      <div>
        <nav class="mb-0.5 flex items-center gap-2 text-xs text-gray-500">
          <a href="/" class="hover:text-gray-700">Home</a>
          <span>/</span>
          <span class="text-gray-700">Skills</span>
        </nav>
        <h1 class="text-lg font-bold text-gray-900">Skill Library</h1>
      </div>
      <a
        href="/agents"
        class="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
      >
        Agent Registry
      </a>
    </div>
  </header>

  <main class="mx-auto max-w-3xl px-4 py-8">
    {#if error}
      <div
        class="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
      >
        {error}
      </div>
    {/if}

    {#if $agentRegistryStore.loading && $agentRegistryStore.skills.length === 0}
      <p class="text-sm text-gray-400">Loading...</p>
    {:else}
      <SkillManager agents={$agentRegistryStore.agents} />
    {/if}
  </main>
</div>
