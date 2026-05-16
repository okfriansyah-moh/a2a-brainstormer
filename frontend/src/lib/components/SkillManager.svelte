<script lang="ts">
  import type {
    Skill,
    Agent,
    CreateSkillRequest,
    UpdateSkillRequest,
  } from "$lib/types";
  import {
    createSkill,
    updateSkill,
    deleteSkill,
    attachSkill,
    detachSkill,
  } from "$lib/services/api";
  import { agentRegistryStore } from "$lib/stores/agentRegistryStore";

  /** Full agent list (for attachment UI). */
  export let agents: Agent[] = [];

  let error = "";
  let successMessage = "";

  // ── Form state ──────────────────────────────────────────────────────────
  let showForm = false;
  let editingSkill: Skill | null = null;

  let formName = "";
  let formDescription = "";
  let formPrompt = "";

  function openCreateForm() {
    editingSkill = null;
    formName = "";
    formDescription = "";
    formPrompt = "";
    showForm = true;
    error = "";
    successMessage = "";
  }

  function openEditForm(skill: Skill) {
    editingSkill = skill;
    formName = skill.name;
    formDescription = skill.description;
    formPrompt = skill.prompt;
    showForm = true;
    error = "";
    successMessage = "";
  }

  function cancelForm() {
    showForm = false;
    editingSkill = null;
    error = "";
  }

  $: formValid = formName.trim().length > 0 && formPrompt.trim().length > 0;

  async function handleSubmitForm() {
    if (!formValid) return;
    error = "";
    successMessage = "";
    agentRegistryStore.setLoading(true);
    try {
      if (editingSkill) {
        const req: UpdateSkillRequest = {
          name: formName.trim(),
          description: formDescription.trim(),
          prompt: formPrompt.trim(),
        };
        const updated = await updateSkill(editingSkill.id, req);
        agentRegistryStore.updateSkill(updated);
        successMessage = `Skill "${updated.name}" updated.`;
      } else {
        const req: CreateSkillRequest = {
          name: formName.trim(),
          description: formDescription.trim(),
          prompt: formPrompt.trim(),
        };
        const created = await createSkill(req);
        agentRegistryStore.addSkill(created);
        successMessage = `Skill "${created.name}" created.`;
      }
      showForm = false;
      editingSkill = null;
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to save skill.";
    } finally {
      agentRegistryStore.setLoading(false);
    }
  }

  let deletingId = "";

  async function handleDelete(skill: Skill) {
    if (!confirm(`Delete skill "${skill.name}"? This cannot be undone.`))
      return;
    deletingId = skill.id;
    error = "";
    successMessage = "";
    try {
      await deleteSkill(skill.id);
      agentRegistryStore.removeSkill(skill.id);
      successMessage = `Skill "${skill.name}" deleted.`;
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to delete skill.";
    } finally {
      deletingId = "";
    }
  }

  // ── Attachment logic ────────────────────────────────────────────────────
  /** Returns true if the given agent has the given skill attached. */
  function agentHasSkill(agent: Agent, skillId: string): boolean {
    return agent.skills.some((s) => s.id === skillId);
  }

  let togglingAttach: Record<string, boolean> = {}; // key: `${agentId}:${skillId}`

  async function toggleAttachment(agent: Agent, skill: Skill) {
    const key = `${agent.id}:${skill.id}`;
    if (togglingAttach[key]) return;
    togglingAttach = { ...togglingAttach, [key]: true };
    error = "";
    try {
      const has = agentHasSkill(agent, skill.id);
      if (has) {
        await detachSkill(agent.id, skill.id);
        const updated: Agent = {
          ...agent,
          skills: agent.skills.filter((s) => s.id !== skill.id),
        };
        agentRegistryStore.updateAgent(updated);
      } else {
        await attachSkill(agent.id, skill.id);
        const updated: Agent = { ...agent, skills: [...agent.skills, skill] };
        agentRegistryStore.updateAgent(updated);
      }
    } catch (err) {
      error =
        err instanceof Error
          ? err.message
          : "Failed to update skill attachment.";
    } finally {
      togglingAttach = Object.fromEntries(
        Object.entries(togglingAttach).filter(([k]) => k !== key),
      );
    }
  }
</script>

<div class="space-y-4">
  <!-- Toolbar -->
  <div class="flex items-center justify-between">
    <h2 class="text-base font-semibold text-gray-900">Skills</h2>
    <button
      class="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700"
      on:click={openCreateForm}
    >
      + New Skill
    </button>
  </div>

  <!-- Feedback -->
  {#if error}
    <div
      class="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700"
    >
      {error}
    </div>
  {/if}
  {#if successMessage}
    <div
      class="rounded-md border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-700"
    >
      {successMessage}
    </div>
  {/if}

  <!-- Create/edit form -->
  {#if showForm}
    <div class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
      <h3 class="mb-3 text-sm font-semibold text-gray-900">
        {editingSkill ? "Edit Skill" : "New Skill"}
      </h3>
      <div class="space-y-3">
        <div>
          <label
            for="skill-name"
            class="mb-1 block text-xs font-medium text-gray-700">Name *</label
          >
          <input
            id="skill-name"
            type="text"
            class="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
            bind:value={formName}
            placeholder="Security Review"
            maxlength={100}
          />
        </div>
        <div>
          <label
            for="skill-description"
            class="mb-1 block text-xs font-medium text-gray-700"
            >Description</label
          >
          <input
            id="skill-description"
            type="text"
            class="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
            bind:value={formDescription}
            placeholder="What this skill makes the agent do..."
            maxlength={500}
          />
        </div>
        <div>
          <label
            for="skill-prompt"
            class="mb-1 block text-xs font-medium text-gray-700"
          >
            Prompt Fragment *
            <span class="font-normal text-gray-400">
              (appended to the agent's system prompt when this skill is active)
            </span>
          </label>
          <textarea
            id="skill-prompt"
            class="w-full resize-y rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
            rows={5}
            bind:value={formPrompt}
            placeholder="When reviewing architecture decisions, always consider OWASP Top 10..."
          ></textarea>
        </div>
      </div>
      <div class="mt-3 flex gap-2">
        <button
          type="button"
          class="rounded-md bg-blue-600 px-4 py-1.5 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          disabled={!formValid || $agentRegistryStore.loading}
          on:click={handleSubmitForm}
        >
          {editingSkill ? "Save Changes" : "Create Skill"}
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

  <!-- Skill list -->
  {#if $agentRegistryStore.skills.length === 0 && !showForm}
    <div
      class="rounded-xl border border-dashed border-gray-300 bg-white p-10 text-center"
    >
      <p class="text-sm text-gray-400 italic">
        No skills yet. Create one to inject specialized behaviors into agents.
      </p>
    </div>
  {:else}
    <div class="space-y-3">
      {#each $agentRegistryStore.skills as skill}
        <div class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm">
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0 flex-1">
              <p class="text-sm font-semibold text-gray-900">{skill.name}</p>
              {#if skill.description}
                <p class="text-xs text-gray-500">{skill.description}</p>
              {/if}
              <p class="mt-1 text-xs text-gray-400 line-clamp-2 font-mono">
                {skill.prompt}
              </p>

              <!-- Agent attachment list -->
              {#if agents.length > 0}
                <div class="mt-2">
                  <p class="mb-1 text-xs font-medium text-gray-600">
                    Attached to agents:
                  </p>
                  <div class="flex flex-wrap gap-1">
                    {#each agents as agent}
                      {@const attached = agentHasSkill(agent, skill.id)}
                      {@const key = `${agent.id}:${skill.id}`}
                      <button
                        type="button"
                        class="rounded-full border px-2 py-0.5 text-xs transition-colors {attached
                          ? 'border-blue-300 bg-blue-100 text-blue-800 hover:bg-blue-200'
                          : 'border-gray-200 bg-gray-50 text-gray-500 hover:bg-gray-100'} disabled:opacity-50"
                        disabled={!!togglingAttach[key]}
                        title={attached
                          ? `Detach from ${agent.name}`
                          : `Attach to ${agent.name}`}
                        on:click={() => toggleAttachment(agent, skill)}
                      >
                        {#if togglingAttach[key]}
                          ...
                        {:else}
                          {agent.name}
                          {attached ? "✓" : "+"}
                        {/if}
                      </button>
                    {/each}
                  </div>
                </div>
              {/if}
            </div>
            <div class="flex shrink-0 gap-2">
              <button
                class="rounded border border-gray-300 px-3 py-1 text-xs text-gray-600 hover:bg-gray-50"
                on:click={() => openEditForm(skill)}
              >
                Edit
              </button>
              <button
                class="rounded border border-red-200 px-3 py-1 text-xs text-red-600 hover:bg-red-50 disabled:opacity-40"
                disabled={deletingId === skill.id}
                on:click={() => handleDelete(skill)}
              >
                {deletingId === skill.id ? "Deleting..." : "Delete"}
              </button>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>
