<script lang="ts">
  import { onMount } from "svelte";
  import { agentRegistryStore } from "$lib/stores/agentRegistryStore";
  import {
    getAgents,
    getSkills,
    createAgent,
    updateAgent,
    deleteAgent,
  } from "$lib/services/api";
  import type {
    Agent,
    CreateAgentRequest,
    UpdateAgentRequest,
    LLMConfig,
  } from "$lib/types";

  const roleOptions = ["build", "review", "refine", "devils_advocate"];
  const providerOptions = ["copilot", "claude"];

  let error = "";
  let successMessage = "";

  // ── Form state ────────────────────────────────────────────────────────────
  let showForm = false;
  let editingAgent: Agent | null = null;

  let formName = "";
  let formDescription = "";
  let formRole = "build";
  let formPrompt = "";
  let formEndpoint = "";
  let formProvider = "copilot";
  let formModel = "gpt-4o";
  let formCredentialRef = "COPILOT_API_KEY";

  function openCreateForm() {
    editingAgent = null;
    formName = "";
    formDescription = "";
    formRole = "build";
    formPrompt = "";
    formEndpoint = "";
    formProvider = "copilot";
    formModel = "gpt-4o";
    formCredentialRef = "COPILOT_API_KEY";
    showForm = true;
    error = "";
    successMessage = "";
  }

  function openEditForm(agent: Agent) {
    editingAgent = agent;
    formName = agent.name;
    formDescription = agent.description;
    formRole = agent.default_role;
    formPrompt = agent.system_prompt;
    formEndpoint = agent.endpoint;
    formProvider = agent.llm_config.provider;
    formModel = agent.llm_config.model;
    formCredentialRef = agent.llm_config.credential_ref;
    showForm = true;
    error = "";
    successMessage = "";
  }

  function cancelForm() {
    showForm = false;
    editingAgent = null;
    error = "";
  }

  $: formValid =
    formName.trim().length > 0 &&
    formPrompt.trim().length > 0 &&
    formEndpoint.trim().length > 0 &&
    formModel.trim().length > 0 &&
    formCredentialRef.trim().length > 0;

  async function handleSubmitForm() {
    if (!formValid) return;
    error = "";
    successMessage = "";
    agentRegistryStore.setLoading(true);
    try {
      const llmCfg: LLMConfig = {
        provider: formProvider,
        model: formModel.trim(),
        credential_ref: formCredentialRef.trim(),
      };
      if (editingAgent) {
        const req: UpdateAgentRequest = {
          name: formName.trim(),
          description: formDescription.trim(),
          default_role: formRole,
          system_prompt: formPrompt.trim(),
          endpoint: formEndpoint.trim(),
          llm_config: llmCfg,
        };
        const updated = await updateAgent(editingAgent.id, req);
        agentRegistryStore.updateAgent({
          ...updated,
          skills: editingAgent.skills,
        });
        successMessage = `Agent "${updated.name}" updated.`;
      } else {
        const req: CreateAgentRequest = {
          name: formName.trim(),
          description: formDescription.trim(),
          default_role: formRole,
          system_prompt: formPrompt.trim(),
          endpoint: formEndpoint.trim(),
          llm_config: llmCfg,
        };
        const created = await createAgent(req);
        agentRegistryStore.addAgent({ ...created, skills: [] });
        successMessage = `Agent "${created.name}" created.`;
      }
      showForm = false;
      editingAgent = null;
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to save agent.";
    } finally {
      agentRegistryStore.setLoading(false);
    }
  }

  let deletingId = "";

  async function handleDelete(agent: Agent) {
    if (!confirm(`Delete agent "${agent.name}"? This cannot be undone.`))
      return;
    deletingId = agent.id;
    error = "";
    successMessage = "";
    try {
      await deleteAgent(agent.id);
      agentRegistryStore.removeAgent(agent.id);
      successMessage = `Agent "${agent.name}" deleted.`;
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to delete agent.";
    } finally {
      deletingId = "";
    }
  }

  onMount(async () => {
    agentRegistryStore.setLoading(true);
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
          <span class="text-gray-700">Agents</span>
        </nav>
        <h1 class="text-lg font-bold text-gray-900">Agent Registry</h1>
      </div>
      <div class="flex items-center gap-2">
        <a
          href="/skills"
          class="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
        >
          Skill Library
        </a>
        <button
          class="rounded-md bg-blue-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-blue-700"
          on:click={openCreateForm}
        >
          + New Agent
        </button>
      </div>
    </div>
  </header>

  <main class="mx-auto max-w-4xl px-4 py-8">
    <!-- Feedback messages -->
    {#if error}
      <div
        class="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
      >
        {error}
      </div>
    {/if}
    {#if successMessage}
      <div
        class="mb-4 rounded-md border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-700"
      >
        {successMessage}
      </div>
    {/if}

    <!-- Create/edit form -->
    {#if showForm}
      <div
        class="mb-6 rounded-xl border border-gray-200 bg-white p-6 shadow-sm"
      >
        <h2 class="mb-4 text-base font-semibold text-gray-900">
          {editingAgent ? "Edit Agent" : "New Agent"}
        </h2>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label
              for="agent-name"
              class="mb-1 block text-xs font-medium text-gray-700">Name *</label
            >
            <input
              id="agent-name"
              type="text"
              class="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
              bind:value={formName}
              placeholder="Agent Alpha"
              maxlength={100}
            />
          </div>
          <div>
            <label
              for="agent-role"
              class="mb-1 block text-xs font-medium text-gray-700"
              >Default Role *</label
            >
            <select
              id="agent-role"
              class="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
              bind:value={formRole}
            >
              {#each roleOptions as role}
                <option value={role}>{role}</option>
              {/each}
            </select>
          </div>
          <div class="col-span-2">
            <label
              for="agent-description"
              class="mb-1 block text-xs font-medium text-gray-700"
              >Description</label
            >
            <input
              id="agent-description"
              type="text"
              class="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
              bind:value={formDescription}
              placeholder="What this agent does..."
              maxlength={500}
            />
          </div>
          <div class="col-span-2">
            <label
              for="agent-endpoint"
              class="mb-1 block text-xs font-medium text-gray-700"
              >Endpoint URL *</label
            >
            <input
              id="agent-endpoint"
              type="text"
              class="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
              bind:value={formEndpoint}
              placeholder="http://agent-host:9090"
            />
          </div>
          <!-- LLM config -->
          <div>
            <label
              for="agent-provider"
              class="mb-1 block text-xs font-medium text-gray-700"
              >LLM Provider *</label
            >
            <select
              id="agent-provider"
              class="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
              bind:value={formProvider}
            >
              {#each providerOptions as p}
                <option value={p}>{p}</option>
              {/each}
            </select>
          </div>
          <div>
            <label
              for="agent-model"
              class="mb-1 block text-xs font-medium text-gray-700"
              >Model *</label
            >
            <input
              id="agent-model"
              type="text"
              class="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
              bind:value={formModel}
              placeholder="gpt-4o"
            />
          </div>
          <div class="col-span-2">
            <label
              for="agent-cred"
              class="mb-1 block text-xs font-medium text-gray-700"
            >
              Credential Ref *
              <span class="font-normal text-gray-400"
                >(env var name, e.g. COPILOT_API_KEY)</span
              >
            </label>
            <input
              id="agent-cred"
              type="text"
              class="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
              bind:value={formCredentialRef}
              placeholder="COPILOT_API_KEY"
            />
          </div>
          <div class="col-span-2">
            <label
              for="agent-prompt"
              class="mb-1 block text-xs font-medium text-gray-700"
              >System Prompt *</label
            >
            <textarea
              id="agent-prompt"
              class="w-full resize-y rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
              rows={4}
              bind:value={formPrompt}
              placeholder="You are a specialized design agent..."
            ></textarea>
          </div>
        </div>
        <div class="mt-4 flex gap-2">
          <button
            type="button"
            class="rounded-md bg-blue-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            disabled={!formValid || $agentRegistryStore.loading}
            on:click={handleSubmitForm}
          >
            {editingAgent ? "Save Changes" : "Create Agent"}
          </button>
          <button
            type="button"
            class="rounded-md border border-gray-300 px-4 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
            on:click={cancelForm}
          >
            Cancel
          </button>
        </div>
      </div>
    {/if}

    <!-- Agent list -->
    {#if $agentRegistryStore.loading && $agentRegistryStore.agents.length === 0}
      <p class="text-sm text-gray-400">Loading agents...</p>
    {:else if $agentRegistryStore.agents.length === 0}
      <div
        class="rounded-xl border border-dashed border-gray-300 bg-white p-12 text-center"
      >
        <p class="text-sm text-gray-400 italic">No agents registered yet.</p>
        <button
          class="mt-3 rounded-md bg-blue-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-blue-700"
          on:click={openCreateForm}
        >
          + Register First Agent
        </button>
      </div>
    {:else}
      <div class="space-y-3">
        {#each $agentRegistryStore.agents as agent}
          <div class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
            <div class="flex items-start justify-between gap-4">
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <h3 class="text-sm font-semibold text-gray-900">
                    {agent.name}
                  </h3>
                  <span
                    class="rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-700"
                  >
                    {agent.default_role}
                  </span>
                </div>
                {#if agent.description}
                  <p class="mt-0.5 text-xs text-gray-500">
                    {agent.description}
                  </p>
                {/if}
                <p class="mt-1 text-xs text-gray-400">
                  {agent.llm_config.provider} / {agent.llm_config.model} &middot;
                  <span class="font-mono">{agent.endpoint}</span>
                </p>
                {#if agent.skills.length > 0}
                  <div class="mt-2 flex flex-wrap gap-1">
                    {#each agent.skills as skill}
                      <span
                        class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-600"
                      >
                        {skill.name}
                      </span>
                    {/each}
                  </div>
                {:else}
                  <p class="mt-1 text-xs text-gray-400 italic">
                    No skills attached
                  </p>
                {/if}
              </div>
              <div class="flex shrink-0 gap-2">
                <button
                  class="rounded border border-gray-300 px-3 py-1 text-xs text-gray-600 hover:bg-gray-50"
                  on:click={() => openEditForm(agent)}
                >
                  Edit
                </button>
                <button
                  class="rounded border border-red-200 px-3 py-1 text-xs text-red-600 hover:bg-red-50 disabled:opacity-40"
                  disabled={deletingId === agent.id}
                  on:click={() => handleDelete(agent)}
                >
                  {deletingId === agent.id ? "Deleting..." : "Delete"}
                </button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </main>
</div>
