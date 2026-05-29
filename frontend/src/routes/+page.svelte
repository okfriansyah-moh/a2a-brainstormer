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
  let selectedDocs: string[] = ["architecture", "roadmap"];
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
        output_docs: selectedDocs.length > 0 ? selectedDocs : undefined,
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

    <!-- How iterations work -->
    <div class="info-box">
      <div class="info-box-title">
        <svg
          width="14"
          height="14"
          viewBox="0 0 16 16"
          fill="none"
          aria-hidden="true"
        >
          <circle
            cx="8"
            cy="8"
            r="7.25"
            stroke="currentColor"
            stroke-width="1.5"
          />
          <path
            d="M8 7v5"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
          />
          <circle cx="8" cy="4.5" r="0.75" fill="currentColor" />
        </svg>
        How iterations work
      </div>
      <p class="info-box-body">
        Each session runs up to <strong
          >{maxIterations} iteration{maxIterations !== 1 ? "s" : ""}</strong
        >. Every iteration sends the state through all {selectedAgentIds.length >
        0
          ? selectedAgentIds.length
          : "selected"} agent{selectedAgentIds.length !== 1 ? "s" : ""} in order —
        the output of each agent feeds the next. After each full pass, a
        <strong>convergence check</strong>
        evaluates whether the state has stabilised (confidence delta, risk
        stability, and open-question stability). If all conditions are met, the
        pipeline stops early and marks the session as <em>converged</em>. If
        not, it continues to the next pass. Seeing multiple passes is
        <strong>expected and intentional</strong>
        — earlier passes draft the design, later passes refine and critique it
        until the agents agree. The session will not run beyond
        {maxIterations} iteration{maxIterations !== 1 ? "s" : ""} regardless of convergence.
      </p>
    </div>

    <!-- Documents to generate -->
    <div style="margin-bottom:18px;">
      <div class="field-label">Documents to Generate</div>
      <div style="display:flex;gap:20px;flex-wrap:wrap;">
        {#each [{ key: "architecture", label: "Architecture" }, { key: "roadmap", label: "Roadmap" }, { key: "plan", label: "Plan" }, { key: "readme", label: "README" }] as doc (doc.key)}
          <label
            style="display:flex;align-items:center;gap:6px;cursor:pointer;font-size:0.875rem;"
          >
            <input
              type="checkbox"
              value={doc.key}
              checked={selectedDocs.includes(doc.key)}
              on:change={(e) => {
                if ((e.target as HTMLInputElement).checked) {
                  selectedDocs = [...selectedDocs, doc.key];
                } else {
                  selectedDocs = selectedDocs.filter((k) => k !== doc.key);
                }
              }}
            />
            {doc.label}
          </label>
        {/each}
      </div>
      <p style="font-size:0.72rem;color:var(--ink-300);margin:4px 0 0;">
        Select which documents to generate at finalize time.
      </p>
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

  .info-box {
    border: 1.5px solid #c7d9f5;
    background: #f0f6ff;
    border-radius: 12px;
    padding: 14px 16px;
    margin-bottom: 18px;
  }

  .info-box-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-weight: 700;
    font-size: 0.8125rem;
    color: #1f5fbf;
    margin-bottom: 6px;
  }

  .info-box-body {
    font-size: 0.8125rem;
    color: #2d4a7a;
    line-height: 1.6;
    margin: 0;
  }

  .info-box-body strong {
    font-weight: 700;
    color: #1a3b6e;
  }

  .info-box-body em {
    font-style: normal;
    font-weight: 600;
    color: #1a3b6e;
  }

  @media (max-width: 700px) {
    div[style*="grid-template-columns:1fr 1fr"] {
      grid-template-columns: 1fr !important;
    }
  }
</style>
