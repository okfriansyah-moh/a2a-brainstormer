<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { agentRegistryStore } from "$lib/stores/agentRegistryStore";
  import AgentSelector from "$lib/components/AgentSelector.svelte";
  import { getAgents, createSession } from "$lib/services/api";
  import type { LLMConfig } from "$lib/types";

  let idea = "";
  let selectedAgentIds: string[] = [];
  let roleOverrides: Record<string, string> = {};
  let skillOverrides: Record<string, string[]> = {};
  let modelOverrides: Record<string, string> = {};
  let maxIterations = 5;
  let submitting = false;
  let error = "";

  onMount(async () => {
    agentRegistryStore.setLoading(true);
    try {
      const agents = await getAgents();
      agentRegistryStore.setAgents(agents);
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to load agents.";
    } finally {
      agentRegistryStore.setLoading(false);
    }
  });

  $: tooFewAgents = selectedAgentIds.length === 1;
  $: canSubmit =
    idea.trim().length > 0 &&
    selectedAgentIds.length >= 2 &&
    maxIterations >= 1 &&
    maxIterations <= 20 &&
    !submitting;

  $: estimatedRuntime = (() => {
    const secs = selectedAgentIds.length * maxIterations * 8;
    if (secs < 60) return `~${secs}s`;
    const m = Math.floor(secs / 60);
    const s = secs % 60;
    return s > 0 ? `~${m}m ${s}s` : `~${m}m`;
  })();

  async function handleSubmit() {
    if (!canSubmit) return;
    submitting = true;
    error = "";
    try {
      const llmOverrides: Record<string, Partial<LLMConfig>> = {};
      for (const [agentId, model] of Object.entries(modelOverrides)) {
        if (model.trim()) llmOverrides[agentId] = { model: model.trim() };
      }
      const resolvedRoleOverrides: Record<string, string> | undefined =
        Object.keys(roleOverrides).length > 0 ? roleOverrides : undefined;
      const resolvedSkillOverrides: Record<string, string[]> | undefined =
        Object.keys(skillOverrides).length > 0 ? skillOverrides : undefined;

      const response = await createSession({
        idea: idea.trim(),
        agent_ids: selectedAgentIds,
        max_iterations: maxIterations,
        role_overrides: resolvedRoleOverrides,
        llm_overrides:
          Object.keys(llmOverrides).length > 0 ? llmOverrides : undefined,
        skill_overrides: resolvedSkillOverrides,
      });
      await goto(`/session/${response.id}`);
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to create session.";
    } finally {
      submitting = false;
    }
  }
</script>

<!-- Page body -->
<div class="artboard">
  <div class="panel" style="max-width:920px;margin:0 auto;padding:28px;">
    <h2 style="font-size:1.4rem;margin-bottom:6px;">
      Start New Design Session
    </h2>
    <p style="color:var(--ink-500);font-size:0.875rem;margin:0 0 22px;">
      Turn a raw idea into architecture and roadmap through controlled agent
      iterations.
    </p>

    {#if error}
      <div
        style="border:1px solid #f5c6d0;background:#fff5f7;color:var(--danger);border-radius:10px;padding:10px 14px;font-size:0.875rem;margin-bottom:16px;"
      >
        {error}
      </div>
    {/if}

    <!-- Idea textarea -->
    <div style="margin-bottom:18px;">
      <div class="field-label">Product Idea</div>
      <textarea
        id="idea"
        style="width:100%;border:1px solid #cfd8ea;border-radius:12px;background:#fff;color:var(--ink-900);padding:11px 12px;font:inherit;min-height:104px;resize:none;"
        placeholder="Describe the idea you want to design..."
        bind:value={idea}
        maxlength={4000}
      ></textarea>
      <p
        style="text-align:right;font-size:0.72rem;color:var(--ink-300);margin:3px 0 0;"
      >
        {idea.length}/4000
      </p>
    </div>

    <!-- 2-column grid: iterations + agent pool -->
    <div
      style="display:grid;grid-template-columns:1fr 1fr;gap:14px;margin-bottom:18px;"
    >
      <!-- Left: max iterations -->
      <div>
        <div class="field-label">Max Iterations</div>
        <input
          id="max-iter"
          type="number"
          style="width:100%;border:1px solid #cfd8ea;border-radius:12px;background:#fff;color:var(--ink-900);padding:11px 12px;font:inherit;"
          min="1"
          max="20"
          bind:value={maxIterations}
        />
        <p style="font-size:0.72rem;color:var(--ink-300);margin:4px 0 0;">
          Between 1 and 20
        </p>
      </div>

      <!-- Right: agent pool -->
      <div>
        <div class="field-label">Agent Pool</div>
        <AgentSelector
          agents={$agentRegistryStore.agents}
          loading={$agentRegistryStore.loading}
          bind:selectedAgentIds
          bind:roleOverrides
          bind:skillOverrides
          bind:modelOverrides
          poolMode={true}
        />
        {#if tooFewAgents}
          <p style="font-size:0.72rem;color:var(--danger);margin:4px 0 0;">
            Select at least one more agent (minimum 2 required).
          </p>
        {/if}
      </div>
    </div>

    <!-- CTA row -->
    <div
      style="display:flex;justify-content:space-between;align-items:center;margin-top:4px;"
    >
      <button
        type="button"
        class="btn-primary"
        style="padding:12px 28px;font-size:0.9375rem;"
        disabled={!canSubmit}
        on:click={handleSubmit}
      >
        {#if submitting}Starting session…{:else}Start Session{/if}
      </button>
      <span style="color:var(--ink-500);font-size:0.8125rem;">
        Estimated runtime: {estimatedRuntime}
      </span>
    </div>
  </div>
</div>

<style>
  .field-label {
    font-weight: 600;
    font-size: 0.8125rem;
    margin-bottom: 7px;
    color: var(--ink-900);
  }

  @media (max-width: 700px) {
    div[style*="grid-template-columns:1fr 1fr"] {
      grid-template-columns: 1fr !important;
    }
  }
</style>
