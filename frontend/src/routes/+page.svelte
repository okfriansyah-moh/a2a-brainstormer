<script lang="ts">
  import { onMount } from "svelte";
  import { goto } from "$app/navigation";
  import { agentRegistryStore } from "$lib/stores/agentRegistryStore";
  import AgentSelector from "$lib/components/AgentSelector.svelte";
  import TechConstraints from "$lib/components/TechConstraints.svelte";
  import DiscoveryClarify from "$lib/components/DiscoveryClarify.svelte";
  import { getAgents, createSession } from "$lib/services/api";
  import type {
    DiscoveryAnswers,
    LLMConfig,
    TechConstraints as TechConstraintsType,
  } from "$lib/types";

  type Step = "home" | "clarify";

  let step: Step = "home";
  let idea = "";
  let selectedAgentIds: string[] = [];
  let roleOverrides: Record<string, string> = {};
  let skillOverrides: Record<string, string[]> = {};
  let modelOverrides: Record<string, string> = {};
  let maxIterations = 5;
  let selectedDocs: string[] = ["architecture", "plan"];
  let techConstraints: TechConstraintsType = { agents_decide: true };
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
  $: canProceed =
    idea.trim().length >= 20 &&
    idea.trim().length <= 4000 &&
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

  function goToClarify() {
    if (!canProceed) return;
    step = "clarify";
    error = "";
  }

  async function startSession(answers: DiscoveryAnswers) {
    if (!canProceed) return;
    submitting = true;
    error = "";
    try {
      const llmOverrides: Record<string, Partial<LLMConfig>> = {};
      for (const [agentId, model] of Object.entries(modelOverrides)) {
        if (model.trim()) llmOverrides[agentId] = { model: model.trim() };
      }
      const response = await createSession({
        idea: idea.trim(),
        agent_ids: selectedAgentIds,
        max_iterations: maxIterations,
        output_docs: selectedDocs.length > 0 ? selectedDocs : undefined,
        discovery_answers: answers,
        tech_constraints: techConstraints,
        role_overrides:
          Object.keys(roleOverrides).length > 0 ? roleOverrides : undefined,
        llm_overrides:
          Object.keys(llmOverrides).length > 0 ? llmOverrides : undefined,
        skill_overrides:
          Object.keys(skillOverrides).length > 0 ? skillOverrides : undefined,
      });
      await goto(`/session/${response.id}`);
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to create session.";
    } finally {
      submitting = false;
    }
  }
</script>

<div class="artboard">
  <div class="panel" style="max-width:920px;margin:0 auto;padding:28px;">
    {#if step === "home"}
      <h2 style="font-size:1.4rem;margin-bottom:6px;">Start New Design Session</h2>
      <p style="color:var(--ink-500);font-size:0.875rem;margin:0 0 22px;">
        Turn a raw idea into architecture and plan through controlled agent iterations.
      </p>

      {#if error}
        <div class="feedback-error">{error}</div>
      {/if}

      <div style="margin-bottom:18px;">
        <div class="field-label">Product Idea</div>
        <textarea
          id="idea"
          class="idea-input"
          placeholder="Describe the idea you want to design (min 20 characters)…"
          bind:value={idea}
          maxlength={4000}
        ></textarea>
        <p class="char-count">{idea.length}/4000</p>
      </div>

      <div class="grid-2" style="margin-bottom:18px;">
        <div>
          <div class="field-label">Max Iterations</div>
          <input
            id="max-iter"
            type="number"
            class="text-input"
            min="1"
            max="20"
            bind:value={maxIterations}
          />
        </div>
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
            <p class="hint-error">Select at least one more agent (minimum 2 required).</p>
          {/if}
        </div>
      </div>

      <TechConstraints bind:value={techConstraints} />

      <div class="info-box">
        <div class="info-box-title">How iterations work</div>
        <p class="info-box-body">
          Each session runs up to <strong>{maxIterations}</strong> iteration passes through
          {selectedAgentIds.length || "selected"} agents until convergence or max iterations.
        </p>
      </div>

      <div style="margin-bottom:18px;">
        <div class="field-label">Documents to Generate</div>
        <div class="doc-checkboxes">
          {#each [{ key: "architecture", label: "Architecture" }, { key: "plan", label: "Plan" }, { key: "readme", label: "README" }] as doc (doc.key)}
            <label class="doc-label">
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
      </div>

      <div class="btn-row">
        <button type="button" class="btn-primary" disabled={!canProceed} on:click={goToClarify}>
          Next: Clarify Idea →
        </button>
        <span class="runtime-hint">Estimated runtime: {estimatedRuntime}</span>
      </div>
    {:else}
      {#if error}
        <div class="feedback-error">{error}</div>
      {/if}
      <DiscoveryClarify
        {idea}
        on:back={() => (step = "home")}
        on:submit={(e) => startSession(e.detail)}
      />
      {#if submitting}
        <p class="runtime-hint" style="margin-top:12px;">Starting session…</p>
      {/if}
    {/if}
  </div>
</div>

<style>
  .field-label {
    font-weight: 600;
    font-size: 0.8125rem;
    margin-bottom: 7px;
    color: var(--ink-900);
  }
  .idea-input,
  .text-input {
    width: 100%;
    border: 1px solid #cfd8ea;
    border-radius: 12px;
    background: #fff;
    color: var(--ink-900);
    padding: 11px 12px;
    font: inherit;
  }
  .idea-input {
    min-height: 104px;
    resize: none;
  }
  .char-count {
    text-align: right;
    font-size: 0.72rem;
    color: var(--ink-300);
    margin: 3px 0 0;
  }
  .grid-2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }
  .hint-error {
    font-size: 0.72rem;
    color: var(--danger);
    margin: 4px 0 0;
  }
  .info-box {
    border: 1.5px solid #c7d9f5;
    background: #f0f6ff;
    border-radius: 12px;
    padding: 14px 16px;
    margin-bottom: 18px;
  }
  .info-box-title {
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
  .doc-checkboxes {
    display: flex;
    gap: 20px;
    flex-wrap: wrap;
  }
  .doc-label {
    display: flex;
    align-items: center;
    gap: 6px;
    cursor: pointer;
    font-size: 0.875rem;
  }
  .btn-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 4px;
  }
  .runtime-hint {
    color: var(--ink-500);
    font-size: 0.8125rem;
  }
  .feedback-error {
    border: 1px solid #f5c6d0;
    background: #fff5f7;
    color: var(--danger);
    border-radius: 10px;
    padding: 10px 14px;
    font-size: 0.875rem;
    margin-bottom: 16px;
  }
  @media (max-width: 700px) {
    .grid-2 {
      grid-template-columns: 1fr;
    }
  }
</style>
