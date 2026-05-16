<script lang="ts">
  import type { Agent } from "$lib/types";

  /** Full list of available agents to select from. */
  export let agents: Agent[] = [];

  /** IDs of agents that have been selected. Bindable. */
  export let selectedAgentIds: string[] = [];

  /** Per-agent role overrides. Key = agent id, value = role string. Bindable. */
  export let roleOverrides: Record<string, string> = {};

  /** Per-agent skill overrides. Key = agent id, value = array of skill UUIDs (null = use defaults). Bindable. */
  export let skillOverrides: Record<string, string[]> = {};

  /** Per-agent LLM model override. Key = agent id, value = model string. Bindable. */
  export let modelOverrides: Record<string, string> = {};

  export let loading: boolean = false;

  const roleOptions = ["build", "review", "refine", "devils_advocate"];

  function toggleAgent(id: string) {
    if (selectedAgentIds.includes(id)) {
      selectedAgentIds = selectedAgentIds.filter((a) => a !== id);
      // Clean up overrides when deselected
      roleOverrides = Object.fromEntries(
        Object.entries(roleOverrides).filter(([k]) => k !== id),
      );
      skillOverrides = Object.fromEntries(
        Object.entries(skillOverrides).filter(([k]) => k !== id),
      );
      modelOverrides = Object.fromEntries(
        Object.entries(modelOverrides).filter(([k]) => k !== id),
      );
    } else {
      selectedAgentIds = [...selectedAgentIds, id];
    }
  }

  function toggleSkillOverride(agentId: string, skillId: string) {
    const current = skillOverrides[agentId];
    if (current === undefined) {
      // Was using defaults — switch to explicit override (all defaults minus this one)
      const agent = agents.find((a) => a.id === agentId);
      if (!agent) return;
      const defaults = agent.skills
        .map((s) => s.id)
        .filter((id) => id !== skillId);
      skillOverrides = { ...skillOverrides, [agentId]: defaults };
    } else if (current.includes(skillId)) {
      skillOverrides = {
        ...skillOverrides,
        [agentId]: current.filter((id) => id !== skillId),
      };
    } else {
      skillOverrides = {
        ...skillOverrides,
        [agentId]: [...current, skillId],
      };
    }
  }

  function isSkillActive(agentId: string, skillId: string): boolean {
    const override = skillOverrides[agentId];
    if (override === undefined) return true; // using defaults = all attached skills active
    return override.includes(skillId);
  }

  $: minAgentsSelected = selectedAgentIds.length >= 2;
</script>

{#if loading}
  <p class="text-sm text-gray-400">Loading agents...</p>
{:else if agents.length === 0}
  <p class="text-sm text-gray-400 italic">
    No agents registered.
    <a href="/agents" class="text-blue-600 underline hover:text-blue-800"
      >Add agents</a
    >
    first.
  </p>
{:else}
  <div class="space-y-3">
    {#each agents as agent}
      {@const selected = selectedAgentIds.includes(agent.id)}
      <div
        class="rounded-lg border transition-colors {selected
          ? 'border-blue-400 bg-blue-50'
          : 'border-gray-200 bg-white'}"
      >
        <!-- Agent row header -->
        <label class="flex cursor-pointer items-start gap-3 p-3">
          <input
            type="checkbox"
            class="mt-0.5 h-4 w-4 rounded border-gray-300 text-blue-600"
            checked={selected}
            on:change={() => toggleAgent(agent.id)}
          />
          <div class="min-w-0 flex-1">
            <p class="text-sm font-medium text-gray-900">{agent.name}</p>
            {#if agent.description}
              <p class="truncate text-xs text-gray-500">{agent.description}</p>
            {/if}
            <p class="mt-0.5 text-xs text-gray-400">
              {agent.llm_config.provider} / {agent.llm_config.model} &mdash; default
              role: {agent.default_role}
            </p>
          </div>
        </label>

        <!-- Expanded overrides when selected -->
        {#if selected}
          <div class="border-t border-blue-200 bg-white px-3 pb-3">
            <div class="mt-2 grid grid-cols-2 gap-3">
              <!-- Role override -->
              <div>
                <label
                  for="role-{agent.id}"
                  class="mb-1 block text-xs font-medium text-gray-600"
                  >Role</label
                >
                <select
                  id="role-{agent.id}"
                  class="w-full rounded border border-gray-300 px-2 py-1 text-xs text-gray-800 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
                  value={roleOverrides[agent.id] ?? agent.default_role}
                  on:change={(e) => {
                    roleOverrides = {
                      ...roleOverrides,
                      [agent.id]: e.currentTarget.value,
                    };
                  }}
                >
                  {#each roleOptions as role}
                    <option value={role}>{role}</option>
                  {/each}
                </select>
              </div>

              <!-- Model override -->
              <div>
                <label
                  for="model-{agent.id}"
                  class="mb-1 block text-xs font-medium text-gray-600"
                >
                  Model override <span class="font-normal text-gray-400"
                    >(optional)</span
                  >
                </label>
                <input
                  id="model-{agent.id}"
                  type="text"
                  class="w-full rounded border border-gray-300 px-2 py-1 text-xs text-gray-800 placeholder-gray-400 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
                  placeholder={agent.llm_config.model}
                  value={modelOverrides[agent.id] ?? ""}
                  on:input={(e) => {
                    const v = e.currentTarget.value.trim();
                    if (v) {
                      modelOverrides = { ...modelOverrides, [agent.id]: v };
                    } else {
                      modelOverrides = Object.fromEntries(
                        Object.entries(modelOverrides).filter(
                          ([k]) => k !== agent.id,
                        ),
                      );
                    }
                  }}
                />
              </div>
            </div>

            <!-- Skill toggles -->
            {#if agent.skills.length > 0}
              <div class="mt-2">
                <p class="mb-1 text-xs font-medium text-gray-600">
                  Skills <span class="font-normal text-gray-400"
                    >(deselect to disable for this session)</span
                  >
                </p>
                <div class="flex flex-wrap gap-1">
                  {#each agent.skills as skill}
                    {@const active = isSkillActive(agent.id, skill.id)}
                    <button
                      type="button"
                      class="rounded-full border px-2 py-0.5 text-xs transition-colors {active
                        ? 'border-blue-300 bg-blue-100 text-blue-800'
                        : 'border-gray-200 bg-gray-100 text-gray-400 line-through'}"
                      on:click={() => toggleSkillOverride(agent.id, skill.id)}
                      title={skill.description}
                    >
                      {skill.name}
                    </button>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        {/if}
      </div>
    {/each}
  </div>

  {#if selectedAgentIds.length > 0 && !minAgentsSelected}
    <p class="mt-1 text-xs text-red-500">
      Select at least one more agent (minimum 2 required).
    </p>
  {/if}
{/if}
