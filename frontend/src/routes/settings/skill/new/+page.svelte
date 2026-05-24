<script lang="ts">
  import { goto } from "$app/navigation";
  import { agentRegistryStore } from "$lib/stores/agentRegistryStore";
  import { createSkill } from "$lib/services/api";
  import type { CreateSkillRequest } from "$lib/types";

  // ── Form state ───────────────────────────────────────────────────────────

  let name = "";
  let description = "";
  let prompt = "";

  let submitting = false;
  let error = "";

  // ── Validation ───────────────────────────────────────────────────────────

  $: formValid =
    name.trim() !== "" && description.trim() !== "" && prompt.trim() !== "";

  // ── Submit ───────────────────────────────────────────────────────────────

  async function handleSubmit(): Promise<void> {
    if (!formValid || submitting) return;
    submitting = true;
    error = "";
    try {
      const req: CreateSkillRequest = {
        name: name.trim(),
        description: description.trim(),
        prompt: prompt.trim(),
      };
      const skill = await createSkill(req);
      agentRegistryStore.addSkill(skill);
      await goto("/settings?tab=skills");
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to create skill.";
    } finally {
      submitting = false;
    }
  }
</script>

<div class="artboard">
  <!-- ── Page header ─────────────────────────────────────────────────────── -->
  <div class="form-header">
    <div>
      <div class="form-title">New Skill</div>
      <div class="form-subtitle">
        Define a knowledge package injected into agent system prompts
      </div>
    </div>
    <a
      href="/settings?tab=skills"
      class="back-link"
      on:click={(e) => {
        e.preventDefault();
        goto("/settings?tab=skills");
      }}>← Back to Settings</a
    >
  </div>

  <!-- ── Form panel ─────────────────────────────────────────────────────── -->
  <div class="panel">
    {#if error}
      <div class="feedback-error" role="alert">{error}</div>
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
        placeholder={"## Rules\n1. Always evaluate decisions through a cost and resource efficiency lens.\n2. Flag any architecture choice with >20% cost overhead vs baseline.\n3. Propose cheaper alternatives when available…"}
        bind:value={prompt}
      ></textarea>
    </div>

    <!-- Action buttons -->
    <div class="btn-row">
      <button
        class="btn-primary"
        type="button"
        disabled={!formValid || submitting}
        on:click={handleSubmit}
      >
        {submitting ? "Saving…" : "Save Skill"}
      </button>
      <a
        href="/settings?tab=skills"
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
    .form-header {
      flex-direction: column;
    }
  }
</style>
