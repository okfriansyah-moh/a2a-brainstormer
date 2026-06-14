<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import { agentRegistryStore } from "$lib/stores/agentRegistryStore";
  import { uiStore } from "$lib/stores/uiStore";
  import {
    getAgents,
    getSkills,
    deleteAgent,
    deleteSkill,
    getGlobalLLMConfig,
    updateGlobalLLMConfig,
  } from "$lib/services/api";
  import type { Agent, GlobalLLMConfig, ProviderKind, Skill } from "$lib/types";
  import {
    ALL_PROVIDER_KINDS,
    PROVIDER_MODEL_PLACEHOLDER,
  } from "$lib/types";

  // Tab state driven by URL search param; default to 'agents'
  $: activeTab = $page.url.searchParams.get("tab") ?? "agents";

  $: if (
    activeTab === "global-llm" &&
    globalLLM === null &&
    !globalLLMLoading
  ) {
    void loadGlobalLLM();
  }

  function switchTab(tab: string): void {
    goto(`?tab=${tab}`, { replaceState: true, noScroll: true });
  }

  let error = "";
  let successMessage = "";
  let toastMessage = "";
  let toastTimeout: ReturnType<typeof setTimeout> | null = null;

  async function clearToastQueryParam(): Promise<void> {
    const params = new URLSearchParams($page.url.searchParams);
    params.delete("toast");
    const query = params.toString();
    await goto(query ? `?${query}` : "?tab=agents", {
      replaceState: true,
      noScroll: true,
      keepFocus: true,
    });
  }

  $: {
    const queryToast = $page.url.searchParams.get("toast") ?? "";
    if (queryToast && queryToast !== toastMessage) {
      toastMessage = queryToast;
      if (toastTimeout) {
        clearTimeout(toastTimeout);
      }
      toastTimeout = setTimeout(() => {
        toastMessage = "";
        void clearToastQueryParam();
      }, 2800);
    }
  }

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

  // ── Global LLM state ─────────────────────────────────────────────────────

  let globalLLM: GlobalLLMConfig | null = null;
  let globalLLMError = "";
  let globalLLMLoading = false;
  let globalLLMSaving = false;
  let globalLLMProvider = "deepseek";
  let globalLLMModel = "deepseek-v4-flash";

  const providerOptions: ProviderKind[] = ALL_PROVIDER_KINDS;

  $: modelPlaceholder =
    PROVIDER_MODEL_PLACEHOLDER[globalLLMProvider as ProviderKind] ??
    "e.g. gpt-4o";

  function applyGlobalLLMForm(config: GlobalLLMConfig | null): void {
    const configuredProvider = (config?.provider ?? "deepseek")
      .trim()
      .toLowerCase();
    globalLLMProvider = (providerOptions as string[]).includes(
      configuredProvider,
    )
      ? configuredProvider
      : "deepseek";

    const fallbackModel =
      PROVIDER_MODEL_PLACEHOLDER[globalLLMProvider as ProviderKind] ??
      "deepseek-v4-flash";

    globalLLMModel = config?.model?.trim() || fallbackModel;
  }

  async function loadGlobalLLM(): Promise<void> {
    globalLLMLoading = true;
    globalLLMError = "";
    try {
      globalLLM = await getGlobalLLMConfig();
      applyGlobalLLMForm(globalLLM);
    } catch (err) {
      globalLLMError =
        err instanceof Error
          ? err.message
          : "Failed to load global LLM config.";
    } finally {
      globalLLMLoading = false;
    }
  }

  async function saveGlobalLLM(): Promise<void> {
    globalLLMSaving = true;
    globalLLMError = "";
    try {
      globalLLM = await updateGlobalLLMConfig({
        provider: globalLLMProvider,
        model: globalLLMModel.trim(),
      });
      applyGlobalLLMForm(globalLLM);
      successMessage = "Global LLM defaults updated.";
    } catch (err) {
      globalLLMError =
        err instanceof Error
          ? err.message
          : "Failed to update global LLM config.";
    } finally {
      globalLLMSaving = false;
    }
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

  onDestroy(() => {
    if (toastTimeout) {
      clearTimeout(toastTimeout);
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
      <button
        class="stab"
        class:stab-active={activeTab === "global-llm"}
        type="button"
        on:click={() => switchTab("global-llm")}
      >
        Global LLM
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
          on:click={(e) => {
            e.preventDefault();
            goto("/settings/agent/new");
          }}
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
            on:click={(e) => {
              e.preventDefault();
              goto("/settings/agent/new");
            }}
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
                  <a
                    class="btn-action"
                    href="/settings/agent/{agent.id}"
                    on:click={(e) => {
                      e.preventDefault();
                      goto(`/settings/agent/${agent.id}`);
                    }}
                  >
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
          on:click={(e) => {
            e.preventDefault();
            goto("/settings/skill/new");
          }}
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
            on:click={(e) => {
              e.preventDefault();
              goto("/settings/skill/new");
            }}
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
                  <a
                    class="btn-action"
                    href="/settings/skill/{skill.id}"
                    on:click={(e) => {
                      e.preventDefault();
                      goto(`/settings/skill/${skill.id}`);
                    }}
                  >
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

    <!-- ── Global LLM Tab ───────────────────────────────────────────── -->
    {#if activeTab === "global-llm"}
      {#if globalLLMLoading}
        <p class="loading-msg">Loading global LLM settings…</p>
      {:else if !globalLLM && globalLLMError}
        <div class="feedback-error" role="alert">{globalLLMError}</div>
      {:else if globalLLM}
        <div class="llm-header">
          <div>
            <h3 class="llm-title">Global LLM Configuration</h3>
            <p class="llm-desc">
              Default provider, model, and credential for agents without
              per-agent overrides.
            </p>
          </div>
          {#if globalLLM.available}
            <span class="chip-ok">Credential available</span>
          {:else}
            <span class="chip-err">Missing credential</span>
          {/if}
        </div>

        {#if globalLLMError}
          <div class="feedback-error llm-feedback" role="alert">
            {globalLLMError}
          </div>
        {/if}

        <div class="llm-form">
          <div class="form-grid">
            <div class="field">
              <div class="field-label">Provider</div>
              <select
                class="form-input select-input"
                bind:value={globalLLMProvider}
              >
                {#each providerOptions as p}
                  <option value={p}>{p}</option>
                {/each}
              </select>
            </div>
            <div class="field">
              <div class="field-label">Model</div>
              <input
                class="form-input"
                type="text"
                placeholder={modelPlaceholder}
                bind:value={globalLLMModel}
              />
            </div>
          </div>

          <div class="field credential-field">
            <div class="field-label">API Key</div>
            {#if globalLLM.available}
              <div class="credential-status credential-status-ok">
                Configured via environment. Key values are never stored or
                displayed in this UI.
              </div>
            {:else}
              <div class="credential-status credential-status-missing">
                No API key detected. Add the provider key to your
                <code>.env</code> file and ensure
                <code>GLOBAL_LLM_CREDENTIAL_REF</code> points at the correct
                env var name.
              </div>
            {/if}
          </div>

          <div class="llm-actions">
            <button
              class="btn-primary"
              type="button"
              on:click={saveGlobalLLM}
              disabled={globalLLMSaving}
            >
              {globalLLMSaving ? "Saving…" : "Save Settings"}
            </button>
            <p class="llm-save-note">
              Updates provider and model in the running backend. API keys stay in
              <code>.env</code> only.
            </p>
          </div>
        </div>

        <details class="env-details">
          <summary>Startup environment variables</summary>
          <pre
            class="env-block"
          ><code>GLOBAL_LLM_PROVIDER={globalLLM.provider}
GLOBAL_LLM_MODEL={globalLLM.model}</code></pre>
        </details>
      {/if}
    {/if}
  </div>

  {#if toastMessage}
    <div class="toast-ok" role="status" aria-live="polite">{toastMessage}</div>
  {/if}
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

  .toast-ok {
    position: fixed;
    right: 24px;
    bottom: 24px;
    max-width: min(460px, calc(100vw - 32px));
    padding: 10px 14px;
    border-radius: 10px;
    background: color-mix(in oklab, var(--ok) 86%, black);
    color: #fff;
    border: 1px solid color-mix(in oklab, var(--ok) 70%, black);
    box-shadow: 0 12px 26px rgba(0, 0, 0, 0.18);
    font-size: 13px;
    z-index: 35;
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

  /* ─── Global LLM tab ────────────────────────────────────────────── */
  .llm-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 20px;
  }

  .llm-title {
    margin: 0;
    font-family: "Space Grotesk", sans-serif;
    font-size: 15px;
    font-weight: 700;
    color: var(--ink-900);
  }

  .llm-desc {
    margin: 4px 0 0;
    font-size: 13px;
    color: var(--ink-500);
    max-width: 520px;
    line-height: 1.45;
  }

  .llm-feedback {
    margin-bottom: 16px;
  }

  .llm-form {
    max-width: 640px;
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 18px;
  }

  .field {
    margin-bottom: 18px;
  }

  .form-grid .field {
    margin-bottom: 0;
  }

  .field-label {
    font-weight: 500;
    font-size: 0.8125rem;
    color: var(--ink-700);
    margin-bottom: 6px;
  }

  .field-hint {
    font-size: 0.75rem;
    color: var(--ink-500);
    margin-bottom: 6px;
    line-height: 1.4;
  }

  .form-input {
    width: 100%;
    border: 1.5px solid var(--line);
    border-radius: 8px;
    padding: 9px 12px;
    font-size: 0.875rem;
    background: rgba(255, 255, 255, 0.8);
    outline: none;
    transition: border-color 0.15s;
    color: var(--ink-900);
  }

  .form-input:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 3px rgba(11, 182, 217, 0.1);
  }

  .credential-status {
    font-size: 0.8125rem;
    line-height: 1.45;
    padding: 10px 12px;
    border-radius: 8px;
  }

  .credential-status code {
    font-family: "IBM Plex Mono", monospace;
    font-size: 0.75rem;
    padding: 1px 5px;
    border-radius: 4px;
    background: rgba(255, 255, 255, 0.55);
    border: 1px solid var(--line);
  }

  .credential-status-ok {
    color: var(--ok-ink);
    background: var(--ok-bg-soft);
    border: 1px solid var(--ok-line);
  }

  .credential-status-missing {
    color: var(--ink-700);
    background: rgba(206, 49, 88, 0.05);
    border: 1px solid rgba(206, 49, 88, 0.2);
  }

  .credential-field {
    margin-bottom: 0;
  }

  .select-input {
    cursor: pointer;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='8' viewBox='0 0 12 8'%3E%3Cpath d='M1 1l5 5 5-5' stroke='%235a6282' stroke-width='1.5' fill='none' stroke-linecap='round'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 12px center;
    padding-right: 32px;
    appearance: none;
  }

  .llm-actions {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
    margin-top: 4px;
    padding-top: 4px;
  }

  .llm-save-note {
    margin: 0;
    font-size: 12px;
    color: var(--ink-500);
    line-height: 1.45;
    max-width: 480px;
  }

  .llm-save-note code {
    font-size: 11px;
    padding: 1px 5px;
    border-radius: 4px;
    background: var(--bg-1);
    border: 1px solid var(--line);
  }

  .chip-err {
    display: inline-flex;
    align-items: center;
    flex-shrink: 0;
    padding: 4px 10px;
    border-radius: 20px;
    font-size: 11px;
    font-weight: 600;
    white-space: nowrap;
    background: rgba(206, 49, 88, 0.1);
    color: var(--danger);
    border: 1px solid rgba(206, 49, 88, 0.25);
  }

  .env-details {
    margin-top: 28px;
    max-width: 640px;
    border-top: 1px solid var(--line);
    padding-top: 16px;
  }

  .env-details summary {
    font-size: 12px;
    font-weight: 600;
    color: var(--ink-500);
    cursor: pointer;
    user-select: none;
    list-style: none;
  }

  .env-details summary::-webkit-details-marker {
    display: none;
  }

  .env-details summary::before {
    content: "▸";
    display: inline-block;
    margin-right: 6px;
    transition: transform 0.15s;
  }

  .env-details[open] summary::before {
    transform: rotate(90deg);
  }

  .env-details summary:hover {
    color: var(--ink-700);
  }

  .env-block {
    font-family: "IBM Plex Mono", monospace;
    font-size: 12px;
    background: var(--bg-1);
    border: 1.5px solid var(--line);
    border-radius: 8px;
    padding: 12px 16px;
    color: var(--ink-700);
    white-space: pre;
    overflow-x: auto;
    margin: 10px 0 0;
  }

  .env-block code {
    font-family: inherit;
    font-size: inherit;
  }

  @media (max-width: 640px) {
    .llm-header {
      flex-direction: column;
      align-items: flex-start;
    }

    .form-grid {
      grid-template-columns: 1fr;
    }

    .form-grid .field {
      margin-bottom: 18px;
    }

    .form-grid .field:last-child {
      margin-bottom: 0;
    }

    .toast-ok {
      left: 16px;
      right: 16px;
      bottom: 16px;
      max-width: none;
    }
  }
</style>
