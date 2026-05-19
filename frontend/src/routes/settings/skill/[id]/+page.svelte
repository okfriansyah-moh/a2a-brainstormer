<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import { agentRegistryStore } from "$lib/stores/agentRegistryStore";
  import {
    getSkill,
    getAgents,
    updateSkill,
    deleteSkill,
  } from "$lib/services/api";
  import type { Agent, UpdateSkillRequest } from "$lib/types";

  // ── Helpers ──────────────────────────────────────────────────────────────

  function roleBadgeClass(role: string): string {
    const map: Record<string, string> = {
      build: "badge-build",
      review: "badge-review",
      refine: "badge-refine",
      devils_advocate: "badge-devils-advocate",
    };
    return map[role] ?? "badge-build";
  }

  // ── Route param ──────────────────────────────────────────────────────────

  $: skillId = $page.params.id;

  // ── Form state ───────────────────────────────────────────────────────────

  let name = "";
  let description = "";
  let prompt = "";

  let loading = true;
  let submitting = false;
  let deleting = false;
  let error = "";
  let successMessage = "";

  // ── Validation ───────────────────────────────────────────────────────────

  $: formValid =
    name.trim() !== "" && description.trim() !== "" && prompt.trim() !== "";

  // ── Agents that use this skill (read-only display) ───────────────────────

  $: attachedAgents = $agentRegistryStore.agents.filter((a: Agent) =>
    a.skills.some((s) => s.id === skillId),
  );

  // ── Submit ───────────────────────────────────────────────────────────────

  async function handleSubmit(): Promise<void> {
    if (!skillId || !formValid || submitting) return;
    submitting = true;
    error = "";
    successMessage = "";
    try {
      const req: UpdateSkillRequest = {
        name: name.trim(),
        description: description.trim(),
        prompt: prompt.trim(),
      };
      const updated = await updateSkill(skillId, req);
      agentRegistryStore.updateSkill(updated);
      successMessage = `Skill "${updated.name}" saved.`;
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to update skill.";
    } finally {
      submitting = false;
    }
  }

  // ── Delete ───────────────────────────────────────────────────────────────

  async function handleDelete(): Promise<void> {
    if (!skillId) return;
    if (!confirm(`Delete skill "${name}"? This cannot be undone.`)) return;
    deleting = true;
    error = "";
    try {
      await deleteSkill(skillId);
      agentRegistryStore.removeSkill(skillId);
      await goto("/settings?tab=skills");
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to delete skill.";
      deleting = false;
    }
  }

  // ── Load ─────────────────────────────────────────────────────────────────

  onMount(async () => {
    if (!skillId) {
      error = "Invalid skill ID.";
      loading = false;
      return;
    }
    loading = true;
    error = "";
    try {
      // Load skill and agent list in parallel (agents needed for "Attached Agents")
      const [skill, fetchedAgents] = await Promise.all([
        getSkill(skillId),
        $agentRegistryStore.agents.length === 0
          ? getAgents()
          : Promise.resolve([] as Agent[]),
      ]);

      if (fetchedAgents.length > 0) {
        agentRegistryStore.setAgents(fetchedAgents);
      }

      name = skill.name;
      description = skill.description;
      prompt = skill.prompt;
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to load skill.";
    } finally {
      loading = false;
    }
  });
</script>

<div class="artboard">
  <!-- ── Page header ─────────────────────────────────────────────────────── -->
  <div class="form-header">
    <div>
      <div class="form-title">Edit Skill</div>
      <div class="form-subtitle">
        Update this knowledge package for agent system prompts
      </div>
    </div>
    <a href="/settings?tab=skills" class="back-link">← Back to Settings</a>
  </div>

  <!-- ── Form panel ─────────────────────────────────────────────────────── -->
  <div class="panel">
    {#if loading}
      <div class="loading-state">Loading skill…</div>
    {:else}
      {#if error}
        <div class="feedback-error" role="alert">{error}</div>
      {/if}
      {#if successMessage}
        <div class="feedback-success" role="status">{successMessage}</div>
      {/if}

      <!-- Skill Name -->
      <div class="field">
        <div class="field-label">Skill Name</div>
        <div class="field-hint">Use kebab-case (e.g. cost-optimization)</div>
        <input
          class="form-input"
          type="text"
          placeholder="e.g. cost-optimization"
          bind:value={name}
        />
      </div>

      <!-- Description -->
      <div class="field">
        <div class="field-label">Description</div>
        <input
          class="form-input"
          type="text"
          placeholder="One sentence — when should this skill be loaded?"
          bind:value={description}
        />
      </div>

      <!-- Prompt Fragment -->
      <div class="field">
        <div class="field-label">Prompt Fragment</div>
        <div class="field-hint">
          This text is appended to the agent's system prompt when the skill is
          active.
        </div>
        <textarea
          class="form-input form-textarea"
          rows="8"
          placeholder="## Rules&#10;1. ..."
          bind:value={prompt}
        ></textarea>
      </div>

      <!-- Attached Agents (read-only) -->
      <div class="field">
        <div class="field-label">Attached Agents</div>
        {#if attachedAgents.length === 0}
          <div class="agents-used-empty">
            No agents currently use this skill.
          </div>
        {:else}
          <div class="agents-used">
            {#each attachedAgents as agent (agent.id)}
              <div class="agent-row">
                <span class="agent-name">{agent.name}</span>
                <span class={roleBadgeClass(agent.default_role)}
                  >{agent.default_role}</span
                >
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Save / Cancel -->
      <div class="btn-row">
        <button
          class="btn-primary"
          type="button"
          disabled={!formValid || submitting}
          on:click={handleSubmit}
        >
          {submitting ? "Saving…" : "Save Changes"}
        </button>
        <a
          href="/settings?tab=skills"
          class="btn-ghost"
          style="display:inline-flex;align-items:center;text-decoration:none;"
        >
          Cancel
        </a>
      </div>

      <!-- Danger zone -->
      <div class="danger-zone">
        <div class="danger-title">Delete Skill</div>
        <p class="danger-desc">
          Removing this skill is permanent. Agents currently using it will lose
          the skill attachment.
        </p>
        <button
          class="btn-danger"
          type="button"
          disabled={deleting}
          on:click={handleDelete}
        >
          {deleting ? "Deleting…" : "Delete Skill"}
        </button>
      </div>
    {/if}
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

  .loading-state {
    padding: 40px 0;
    text-align: center;
    color: var(--ink-300);
    font-size: 0.875rem;
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
    min-height: 160px;
    font-family: "IBM Plex Mono", monospace;
    font-size: 0.8125rem;
    line-height: 1.6;
  }

  .agents-used {
    border: 1.5px solid var(--line);
    border-radius: 10px;
    background: rgba(255, 255, 255, 0.5);
    overflow: hidden;
  }

  .agent-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 14px;
    border-bottom: 1px solid var(--line);
  }

  .agent-row:last-child {
    border-bottom: none;
  }

  .agent-name {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--ink-900);
    flex: 1;
  }

  .agents-used-empty {
    padding: 14px 16px;
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
    margin-bottom: 32px;
  }

  .danger-zone {
    border: 1.5px solid rgba(206, 49, 88, 0.25);
    border-radius: 10px;
    padding: 18px 20px;
    background: rgba(206, 49, 88, 0.03);
  }

  .danger-title {
    font-weight: 600;
    font-size: 0.875rem;
    color: var(--danger);
    margin-bottom: 6px;
  }

  .danger-desc {
    font-size: 0.8125rem;
    color: var(--ink-500);
    margin: 0 0 14px 0;
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

  .feedback-success {
    background: rgba(27, 159, 102, 0.08);
    border: 1px solid rgba(27, 159, 102, 0.25);
    border-radius: 8px;
    padding: 10px 14px;
    color: var(--ok);
    font-size: 0.875rem;
    margin-bottom: 20px;
  }

  @media (max-width: 640px) {
    .form-header {
      flex-direction: column;
    }
  }
</style>
