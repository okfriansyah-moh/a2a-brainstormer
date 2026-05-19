<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import { agentRegistryStore } from "$lib/stores/agentRegistryStore";
  import { uiStore } from "$lib/stores/uiStore";
  import {
    getAgents,
    getSkills,
    deleteAgent,
    deleteSkill,
  } from "$lib/services/api";
  import type { Agent, Skill } from "$lib/types";

  // Tab state driven by URL search param; default to 'agents'
  $: activeTab = $page.url.searchParams.get("tab") ?? "agents";

  function switchTab(tab: string): void {
    goto(`?tab=${tab}`, { replaceState: true, noScroll: true });
  }

  let error = "";
  let successMessage = "";

  // ── Agent actions ─────────────────────────────────────────────────────────

  let deletingAgentId = "";

  async function handleDeleteAgent(agent: Agent): Promise<void> {
    uiStore.openModal({
      title: `Delete "${agent.name}"?`,
      body: "Removing this agent is permanent and cannot be undone.",
      confirmLabel: "Delete Agent",
      confirmDanger: true,
      onConfirm: async () => {
        deletingAgentId = agent.id;
        error = "";
        successMessage = "";
        try {
          await deleteAgent(agent.id);
          agentRegistryStore.removeAgent(agent.id);
          successMessage = `Agent "${agent.name}" deleted.`;
        } catch (err) {
          error =
            err instanceof Error ? err.message : "Failed to delete agent.";
        } finally {
          deletingAgentId = "";
        }
      },
    });
  }

  // ── Skill actions ─────────────────────────────────────────────────────────

  let deletingSkillId = "";

  async function handleDeleteSkill(skill: Skill): Promise<void> {
    uiStore.openModal({
      title: `Delete "${skill.name}"?`,
      body: "Removing this skill is permanent. Any agents using it will lose this skill.",
      confirmLabel: "Delete Skill",
      confirmDanger: true,
      onConfirm: async () => {
        deletingSkillId = skill.id;
        error = "";
        successMessage = "";
        try {
          await deleteSkill(skill.id);
          agentRegistryStore.removeSkill(skill.id);
          successMessage = `Skill "${skill.name}" deleted.`;
        } catch (err) {
          error =
            err instanceof Error ? err.message : "Failed to delete skill.";
        } finally {
          deletingSkillId = "";
        }
      },
    });
  }

  // ── Role badge CSS class ──────────────────────────────────────────────────

  function roleBadgeClass(role: string): string {
    const map: Record<string, string> = {
      build: "badge-build",
      review: "badge-review",
      refine: "badge-refine",
      devils_advocate: "badge-devils-advocate",
    };
    return map[role] ?? "badge-build";
  }

  // ── Count agents that use a given skill ──────────────────────────────────

  function agentCountForSkill(skill: Skill): number {
    return $agentRegistryStore.agents.filter((a) =>
      a.skills.some((s) => s.id === skill.id),
    ).length;
  }

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

<div class="artboard">
  <!-- ── Page header ───────────────────────────────────────────────────── -->
  <div class="settings-header">
    <div>
      <div class="settings-title">Settings</div>
      <div class="settings-subtitle">
        Manage agents, skills, and roles for the brainstorm pipeline
      </div>
    </div>
    <nav class="topbar-nav">
      <a href="/" class="topbar-link">New Session</a>
      <a href="/history" class="topbar-link">Sessions</a>
    </nav>
  </div>

  <!-- ── Content panel ────────────────────────────────────────────────── -->
  <div class="panel settings-panel">
    <!-- Feedback -->
    {#if error}
      <div class="feedback-error" role="alert">{error}</div>
    {/if}
    {#if successMessage}
      <div class="feedback-ok" role="status">{successMessage}</div>
    {/if}

    <!-- Tab navigation -->
    <div class="settings-tabs">
      <button
        class="stab"
        class:stab-active={activeTab === "agents"}
        type="button"
        on:click={() => switchTab("agents")}
      >
        Agents
      </button>
      <button
        class="stab"
        class:stab-active={activeTab === "skills"}
        type="button"
        on:click={() => switchTab("skills")}
      >
        Skills
      </button>
      <button
        class="stab"
        class:stab-active={activeTab === "roles"}
        type="button"
        on:click={() => switchTab("roles")}
      >
        Roles
      </button>
    </div>

    <!-- ── Agents Tab ────────────────────────────────────────────────── -->
    {#if activeTab === "agents"}
      <div class="table-toolbar">
        <h3>Registered Agents</h3>
        <a
          class="btn-primary"
          href="/settings/agent/new"
          style="display:inline-block;text-decoration:none;"
        >
          + New Agent
        </a>
      </div>

      {#if $agentRegistryStore.loading && $agentRegistryStore.agents.length === 0}
        <p class="loading-msg">Loading agents…</p>
      {:else if $agentRegistryStore.agents.length === 0}
        <div class="empty-state">
          <p>No agents registered yet.</p>
          <a
            class="btn-primary"
            href="/settings/agent/new"
            style="display:inline-block;text-decoration:none;margin-top:12px;"
          >
            Register First Agent
          </a>
        </div>
      {:else}
        <table>
          <thead>
            <tr>
              <th>Name</th>
              <th>Role</th>
              <th>Provider / Model</th>
              <th>Skills</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each $agentRegistryStore.agents as agent (agent.id)}
              <tr>
                <td>
                  <strong>{agent.name}</strong>
                  {#if agent.description}
                    <div class="row-sub">{agent.description}</div>
                  {/if}
                </td>
                <td>
                  <span class={roleBadgeClass(agent.default_role)}>
                    {agent.default_role}
                  </span>
                </td>
                <td class="mono-cell">
                  {agent.llm_config.provider} / {agent.llm_config.model}
                </td>
                <td>{agent.skills.length}</td>
                <td><span class="chip-ok">Healthy</span></td>
                <td>
                  <a class="btn-action" href="/settings/agent/{agent.id}">
                    Edit
                  </a>
                  <button
                    class="btn-action btn-delete"
                    type="button"
                    disabled={deletingAgentId === agent.id}
                    on:click={() => handleDeleteAgent(agent)}
                  >
                    {deletingAgentId === agent.id ? "…" : "Delete"}
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    {/if}

    <!-- ── Roles Tab ─────────────────────────────────────────────────── -->
    {#if activeTab === "roles"}
      <div class="table-toolbar">
        <h3>Role Definitions</h3>
      </div>
      <table>
        <thead>
          <tr>
            <th>Role</th>
            <th>Type</th>
            <th>Behavioral Directive</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td><strong>build</strong></td>
            <td><span class="role-type-badge">System</span></td>
            <td class="directive-cell"
              >Propose architecture, draft execution plans, generate initial
              state</td
            >
            <td><span class="protected-label">Protected</span></td>
          </tr>
          <tr>
            <td><strong>review</strong></td>
            <td><span class="role-type-badge">System</span></td>
            <td class="directive-cell"
              >Identify risks, challenge assumptions, add open questions</td
            >
            <td><span class="protected-label">Protected</span></td>
          </tr>
          <tr>
            <td><strong>refine</strong></td>
            <td><span class="role-type-badge">System</span></td>
            <td class="directive-cell"
              >Merge agent outputs, resolve conflicts, improve confidence score</td
            >
            <td><span class="protected-label">Protected</span></td>
          </tr>
          <tr>
            <td><strong>devils_advocate</strong></td>
            <td><span class="role-type-badge">System</span></td>
            <td class="directive-cell"
              >Stress-test decisions, surface edge cases, push back on consensus</td
            >
            <td><span class="protected-label">Protected</span></td>
          </tr>
        </tbody>
      </table>
      <p class="roles-note">
        System roles are protected and cannot be deleted. Assign roles to agents
        when registering or editing them.
      </p>
    {/if}

    <!-- ── Skills Tab ─────────────────────────────────────────────────── -->
    {#if activeTab === "skills"}
      <div class="table-toolbar">
        <h3>Skill Library</h3>
        <a
          class="btn-primary"
          href="/settings/skill/new"
          style="display:inline-block;text-decoration:none;"
        >
          + New Skill
        </a>
      </div>

      {#if $agentRegistryStore.loading && $agentRegistryStore.skills.length === 0}
        <p class="loading-msg">Loading skills…</p>
      {:else if $agentRegistryStore.skills.length === 0}
        <div class="empty-state">
          <p>No skills defined yet.</p>
          <a
            class="btn-primary"
            href="/settings/skill/new"
            style="display:inline-block;text-decoration:none;margin-top:12px;"
          >
            Add First Skill
          </a>
        </div>
      {:else}
        <table>
          <thead>
            <tr>
              <th>Skill</th>
              <th>Description</th>
              <th>Used By</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each $agentRegistryStore.skills as skill (skill.id)}
              <tr>
                <td><strong>{skill.name}</strong></td>
                <td class="desc-cell">{skill.description || "—"}</td>
                <td class="dim-cell">
                  {agentCountForSkill(skill)}
                  {agentCountForSkill(skill) === 1 ? "agent" : "agents"}
                </td>
                <td>
                  <a class="btn-action" href="/settings/skill/{skill.id}">
                    Edit
                  </a>
                  <button
                    class="btn-action btn-delete"
                    type="button"
                    disabled={deletingSkillId === skill.id}
                    on:click={() => handleDeleteSkill(skill)}
                  >
                    {deletingSkillId === skill.id ? "…" : "Delete"}
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    {/if}
  </div>
</div>

<style>
  /* ─── Page header ───────────────────────────────────────────────── */
  .settings-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 20px 0 16px;
  }

  .settings-title {
    font-family: "Space Grotesk", sans-serif;
    font-size: 22px;
    font-weight: 700;
    color: var(--ink-900);
    line-height: 1.2;
  }

  .settings-subtitle {
    font-size: 13px;
    color: var(--ink-500);
    margin-top: 3px;
  }

  /* ─── Panel ─────────────────────────────────────────────────────── */
  .settings-panel {
    padding: 20px 24px;
  }

  /* ─── Tab system ─────────────────────────────────────────────────── */
  .settings-tabs {
    display: flex;
    gap: 2px;
    border-bottom: 1.5px solid var(--line);
    margin-bottom: 20px;
    padding: 0;
  }

  .stab {
    background: none;
    border: none;
    padding: 9px 18px;
    font-family: "IBM Plex Sans", sans-serif;
    font-size: 13px;
    font-weight: 600;
    color: var(--ink-500);
    cursor: pointer;
    border-bottom: 2.5px solid transparent;
    margin-bottom: -1.5px;
    border-radius: 0;
    transition: color 0.15s;
  }

  .stab:hover {
    color: var(--ink-900);
  }

  .stab-active {
    color: var(--accent);
    border-bottom-color: var(--accent);
  }

  /* ─── Table toolbar ─────────────────────────────────────────────── */
  .table-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 14px;
  }

  .table-toolbar h3 {
    margin: 0;
    font-family: "Space Grotesk", sans-serif;
    font-size: 15px;
    font-weight: 700;
    color: var(--ink-900);
  }

  /* ─── Table ─────────────────────────────────────────────────────── */
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 14px;
  }

  th,
  td {
    border-bottom: 1px solid var(--line);
    padding: 10px 8px;
    text-align: left;
  }

  th {
    color: var(--ink-500);
    font-weight: 600;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.4px;
  }

  .desc-cell {
    max-width: 360px;
    color: var(--ink-700);
    font-size: 13px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .dim-cell {
    color: var(--ink-500);
    font-size: 13px;
    white-space: nowrap;
  }

  .mono-cell {
    color: var(--ink-700);
    font-size: 13px;
    font-family: "IBM Plex Mono", monospace;
  }

  .row-sub {
    font-size: 12px;
    color: var(--ink-500);
    margin-top: 2px;
  }

  /* ─── Action buttons ─────────────────────────────────────────────── */
  .btn-action {
    display: inline-block;
    border: 1px solid var(--line);
    background: transparent;
    border-radius: 7px;
    padding: 4px 10px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    color: var(--ink-700);
    margin-right: 4px;
    font-family: "IBM Plex Sans", sans-serif;
    text-decoration: none;
    transition: background 0.12s;
  }

  .btn-action:hover:not(:disabled) {
    background: var(--bg-1);
  }

  .btn-delete {
    color: var(--danger);
    border-color: rgba(206, 49, 88, 0.3);
  }

  .btn-delete:hover:not(:disabled) {
    background: rgba(206, 49, 88, 0.06);
  }

  .btn-delete:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* ─── Feedback ──────────────────────────────────────────────────── */
  .feedback-error {
    margin-bottom: 14px;
    padding: 10px 14px;
    border-radius: 8px;
    background: rgba(206, 49, 88, 0.06);
    border: 1px solid rgba(206, 49, 88, 0.3);
    color: var(--danger);
    font-size: 13px;
  }

  .feedback-ok {
    margin-bottom: 14px;
    padding: 10px 14px;
    border-radius: 8px;
    background: rgba(27, 159, 102, 0.08);
    border: 1px solid rgba(27, 159, 102, 0.3);
    color: var(--ok);
    font-size: 13px;
  }

  /* ─── Loading / empty ───────────────────────────────────────────── */
  .loading-msg {
    color: var(--ink-300);
    font-size: 13px;
    padding: 24px 0;
    margin: 0;
  }

  .empty-state {
    text-align: center;
    padding: 48px 20px;
    color: var(--ink-500);
    font-size: 14px;
  }

  .empty-state p {
    margin: 0 0 4px;
  }

  /* ─── Roles tab ─────────────────────────────────────────────────── */
  .directive-cell {
    color: var(--ink-700);
    font-size: 13px;
    max-width: 400px;
  }

  .role-type-badge {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.4px;
    background: rgba(31, 122, 224, 0.1);
    color: var(--accent-2);
    border: 1px solid rgba(31, 122, 224, 0.2);
  }

  .protected-label {
    font-size: 12px;
    color: var(--ink-300);
    font-style: italic;
  }

  .roles-note {
    font-size: 12px;
    color: var(--ink-500);
    margin-top: 14px;
    margin-bottom: 0;
  }
</style>
