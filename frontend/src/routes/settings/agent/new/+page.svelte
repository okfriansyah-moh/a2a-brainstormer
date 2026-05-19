<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { agentRegistryStore } from "$lib/stores/agentRegistryStore";
  import { getSkills, createAgent, attachSkill } from "$lib/services/api";
  import type { CreateAgentRequest } from "$lib/types";

  // ── Form state ───────────────────────────────────────────────────────────

  let name = "";
  let description = "";
  let defaultRole = "build";
  let provider = "copilot";
  let model = "";
  let endpoint = "";
  let credentialRef = "";
  let systemPrompt = "";

  let selectedSkillIds = new Set<string>();
  let submitting = false;
  let error = "";

  const roleOptions = ["build", "review", "refine", "devils_advocate"];
  const providerOptions = ["copilot", "claude"];

  // ── Validation ───────────────────────────────────────────────────────────

  $: formValid =
    name.trim() !== "" &&
    systemPrompt.trim() !== "" &&
    endpoint.trim() !== "" &&
    model.trim() !== "" &&
    credentialRef.trim() !== "";

  // ── Skill selection ──────────────────────────────────────────────────────

  function toggleSkill(skillId: string): void {
    const next = new Set(selectedSkillIds);
    if (next.has(skillId)) {
      next.delete(skillId);
    } else {
      next.add(skillId);
    }
    selectedSkillIds = next;
  }

  // ── Submit ───────────────────────────────────────────────────────────────

  async function handleSubmit(): Promise<void> {
    if (!formValid || submitting) return;
    submitting = true;
    error = "";
    try {
      const req: CreateAgentRequest = {
        name: name.trim(),
        description: description.trim(),
        default_role: defaultRole,
        system_prompt: systemPrompt.trim(),
        endpoint: endpoint.trim(),
        llm_config: {
          provider,
          model: model.trim(),
          credential_ref: credentialRef.trim(),
        },
      };
      const agent = await createAgent(req);
      await Promise.all(
        [...selectedSkillIds].map((sid) => attachSkill(agent.id, sid)),
      );
      const attachedSkills = $agentRegistryStore.skills.filter((s) =>
        selectedSkillIds.has(s.id),
      );
      agentRegistryStore.addAgent({ ...agent, skills: attachedSkills });
      await goto("/settings?tab=agents");
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to create agent.";
    } finally {
      submitting = false;
    }
  }

  // ── Load skill library ───────────────────────────────────────────────────

  onMount(async () => {
    if ($agentRegistryStore.skills.length === 0) {
      try {
        const skills = await getSkills();
        agentRegistryStore.setSkills(skills);
      } catch {
        // Non-fatal: skill pool may be empty or backend unavailable
      }
    }
  });
</script>

<div class="artboard">
  <!-- ── Page header ─────────────────────────────────────────────────────── -->
  <div class="form-header">
    <div>
      <div class="form-title">New Agent</div>
      <div class="form-subtitle">
        Configure a role-specialized agent for the brainstorm pipeline
      </div>
    </div>
    <a href="/settings?tab=agents" class="back-link">← Back to Settings</a>
  </div>

  <!-- ── Form panel ─────────────────────────────────────────────────────── -->
  <div class="panel">
    {#if error}
      <div class="feedback-error" role="alert">{error}</div>
    {/if}

    <!-- Name + Role -->
    <div class="form-grid">
      <div class="field">
        <div class="field-label">Agent Name</div>
        <input
          class="form-input"
          type="text"
          placeholder="e.g. Atlas"
          bind:value={name}
        />
      </div>
      <div class="field">
        <div class="field-label">Role</div>
        <select class="form-input select-input" bind:value={defaultRole}>
          {#each roleOptions as role}
            <option value={role}>{role}</option>
          {/each}
        </select>
      </div>
    </div>

    <!-- Provider + Model -->
    <div class="form-grid">
      <div class="field">
        <div class="field-label">Provider</div>
        <select class="form-input select-input" bind:value={provider}>
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
          placeholder="e.g. gpt-4o"
          bind:value={model}
        />
      </div>
    </div>

    <!-- Endpoint -->
    <div class="field">
      <div class="field-label">Endpoint URL</div>
      <input
        class="form-input"
        type="url"
        placeholder="https://agent.example.com"
        bind:value={endpoint}
      />
    </div>

    <!-- Description -->
    <div class="field">
      <div class="field-label">
        Description <span class="muted-label">(optional)</span>
      </div>
      <input
        class="form-input"
        type="text"
        placeholder="What this agent specializes in"
        bind:value={description}
      />
    </div>

    <!-- Credential Ref -->
    <div class="field">
      <div class="field-label">Credential Reference</div>
      <div class="field-hint">
        Env var name only — never the raw key value (e.g. COPILOT_API_KEY)
      </div>
      <input
        class="form-input"
        type="text"
        placeholder="e.g. COPILOT_API_KEY"
        bind:value={credentialRef}
        autocomplete="off"
      />
    </div>

    <!-- System Prompt -->
    <div class="field">
      <div class="field-label">System Prompt</div>
      <textarea
        class="form-input form-textarea"
        rows="6"
        placeholder="You are a specialized agent responsible for..."
        bind:value={systemPrompt}
      ></textarea>
    </div>

    <!-- Assign Skills -->
    <div class="field">
      <div class="field-label">Assign Skills</div>
      {#if $agentRegistryStore.skills.length === 0}
        <div class="pool-empty">
          No skills registered. Add skills in Settings → Skills tab.
        </div>
      {:else}
        <div class="skill-pool">
          {#each $agentRegistryStore.skills as skill (skill.id)}
            <label class="skill-pool-row">
              <input
                type="checkbox"
                checked={selectedSkillIds.has(skill.id)}
                on:change={() => toggleSkill(skill.id)}
              />
              <span class="skill-name">{skill.name}</span>
              <span class="skill-desc">{skill.description}</span>
            </label>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Action buttons -->
    <div class="btn-row">
      <button
        class="btn-primary"
        type="button"
        disabled={!formValid || submitting}
        on:click={handleSubmit}
      >
        {submitting ? "Saving…" : "Save Agent"}
      </button>
      <a
        href="/settings?tab=agents"
        class="btn-ghost"
        style="display:inline-flex;align-items:center;text-decoration:none;"
      >
        Cancel
      </a>
    </div>
  </div>
</div>

<style>
  .form-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 20px;
    gap: 16px;
  }

  .form-title {
    font-family: "Space Grotesk", sans-serif;
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--ink-900);
  }

  .form-subtitle {
    font-size: 0.875rem;
    color: var(--ink-500);
    margin-top: 4px;
  }

  .back-link {
    font-size: 0.875rem;
    color: var(--ink-500);
    text-decoration: none;
    white-space: nowrap;
    padding-top: 6px;
    flex-shrink: 0;
  }

  .back-link:hover {
    color: var(--accent-2);
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 18px;
  }

  .field {
    margin-bottom: 18px;
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
  }

  .muted-label {
    font-weight: 400;
    color: var(--ink-300);
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

  .form-textarea {
    resize: vertical;
    min-height: 120px;
  }

  .select-input {
    cursor: pointer;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='8' viewBox='0 0 12 8'%3E%3Cpath d='M1 1l5 5 5-5' stroke='%235a6282' stroke-width='1.5' fill='none' stroke-linecap='round'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 12px center;
    padding-right: 32px;
    appearance: none;
    -webkit-appearance: none;
  }

  .skill-pool {
    border: 1.5px solid var(--line);
    border-radius: 10px;
    max-height: 240px;
    overflow-y: auto;
    background: rgba(255, 255, 255, 0.5);
  }

  .skill-pool-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 14px;
    border-bottom: 1px solid var(--line);
    cursor: pointer;
    transition: background 0.1s;
  }

  .skill-pool-row:last-child {
    border-bottom: none;
  }

  .skill-pool-row:hover {
    background: rgba(11, 182, 217, 0.04);
  }

  .skill-pool-row input[type="checkbox"] {
    width: auto;
    accent-color: var(--accent);
    cursor: pointer;
    flex-shrink: 0;
  }

  .skill-name {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--ink-900);
    flex: 1;
    font-family: "IBM Plex Mono", monospace;
  }

  .skill-desc {
    font-size: 0.75rem;
    color: var(--ink-500);
    text-align: right;
    max-width: 260px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .pool-empty {
    padding: 16px;
    font-size: 0.875rem;
    color: var(--ink-300);
    font-style: italic;
    border: 1.5px solid var(--line);
    border-radius: 10px;
  }

  .btn-row {
    display: flex;
    gap: 10px;
    margin-top: 8px;
  }

  .feedback-error {
    background: rgba(206, 49, 88, 0.08);
    border: 1px solid rgba(206, 49, 88, 0.25);
    border-radius: 8px;
    padding: 10px 14px;
    color: var(--danger);
    font-size: 0.875rem;
    margin-bottom: 20px;
  }

  @media (max-width: 640px) {
    .form-grid {
      grid-template-columns: 1fr;
    }

    .form-header {
      flex-direction: column;
    }
  }
</style>
