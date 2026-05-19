<script lang="ts">
  import type { Agent } from "$lib/types";

  /** Full list of available agents to select from. */
  export let agents: Agent[] = [];

  /** IDs of agents that have been selected. Bindable. */
  export let selectedAgentIds: string[] = [];

  /** Per-agent role overrides. Key = agent id, value = role string. Bindable. */
  export let roleOverrides: Record<string, string> = {};

  /** Per-agent skill overrides. Key = agent id, value = array of skill UUIDs. Bindable. */
  export let skillOverrides: Record<string, string[]> = {};

  /** Per-agent LLM model override. Key = agent id, value = model string. Bindable. */
  export let modelOverrides: Record<string, string> = {};

  export let loading: boolean = false;

  /**
   * poolMode=true renders a compact checkbox-row pool (used on the home page).
   * poolMode=false (default) renders the expanded card layout with override controls.
   */
  export let poolMode: boolean = false;

  const roleOptions = ["build", "review", "refine", "devils_advocate"];

  function toggleAgent(id: string) {
    if (selectedAgentIds.includes(id)) {
      selectedAgentIds = selectedAgentIds.filter((a) => a !== id);
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
    if (override === undefined) return true;
    return override.includes(skillId);
  }

  $: minAgentsSelected = selectedAgentIds.length >= 2;
</script>

{#if loading}
  <p style="font-size:0.8125rem;color:var(--ink-500);padding:10px 0;">
    Loading agents…
  </p>
{:else if agents.length === 0}
  <p
    style="font-size:0.8125rem;color:var(--ink-500);font-style:italic;padding:10px 0;"
  >
    No agents registered.
    <a
      href="/settings?tab=agents"
      style="color:var(--accent);text-decoration:underline;">Add agents</a
    >
    first.
  </p>
{:else if poolMode}
  <!-- ── Compact pool layout (home page) ─────────────────────────────── -->
  <div class="pool" class:pool-invalid={selectedAgentIds.length === 1}>
    {#each agents as agent}
      {@const selected = selectedAgentIds.includes(agent.id)}
      <label class="agent-row" class:agent-row-selected={selected}>
        <input
          type="checkbox"
          checked={selected}
          on:change={() => toggleAgent(agent.id)}
        />
        <div class="agent-row-body">
          <div class="agent-row-name">{agent.name}</div>
          <div class="agent-row-meta">
            {agent.llm_config.provider} / {agent.llm_config.model}
          </div>
        </div>
        <span class="badge-{agent.default_role}"
          >{agent.default_role.replace("_", " ")}</span
        >
      </label>
    {/each}
  </div>
{:else}
  <!-- ── Full card layout with override controls ───────────────────────── -->
  <div class="space-y-3">
    {#each agents as agent}
      {@const selected = selectedAgentIds.includes(agent.id)}
      <div
        class="rounded-lg border transition-colors {selected
          ? 'border-blue-400 bg-blue-50'
          : 'border-gray-200 bg-white'}"
      >
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

        {#if selected}
          <div class="border-t border-blue-200 bg-white px-3 pb-3">
            <div class="mt-2 grid grid-cols-2 gap-3">
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

<style>
  /* ── Pool layout ─────────────────────────────────────────────────────── */
  .pool {
    border: 1px solid #cfd8ea;
    border-radius: 12px;
    overflow: hidden;
    background: #fff;
  }

  .pool-invalid {
    border-color: var(--danger);
  }

  .agent-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    cursor: pointer;
    border-bottom: 1px solid #eef1f8;
    transition: background 0.12s;
  }

  .agent-row:last-child {
    border-bottom: none;
  }

  .agent-row:hover {
    background: #f8f9ff;
  }

  .agent-row-selected {
    background: #f0f4ff;
  }

  .agent-row input[type="checkbox"] {
    width: 15px;
    height: 15px;
    accent-color: var(--accent);
    flex-shrink: 0;
  }

  .agent-row-body {
    flex: 1;
    min-width: 0;
  }

  .agent-row-name {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--ink-900);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .agent-row-meta {
    font-size: 0.72rem;
    color: var(--ink-500);
    margin-top: 1px;
  }
</style>
