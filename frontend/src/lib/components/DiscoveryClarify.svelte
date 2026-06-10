<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte";
  import {
    STATIC_DISCOVERY_HINTS,
    mergeHints,
    type DiscoveryHints,
  } from "./DiscoveryChipHints";
  import { fetchDiscoveryHints } from "$lib/services/api";
  import type { DiscoveryAnswers } from "$lib/types";

  export let idea = "";

  const dispatch = createEventDispatcher<{
    back: void;
    submit: DiscoveryAnswers;
  }>();

  let hints: DiscoveryHints = { ...STATIC_DISCOVERY_HINTS };
  let answers: DiscoveryAnswers = { q1: "", q2: [], q3: [], q4: [] };
  let skipped: Record<string, boolean> = {};
  let showOther: Record<string, boolean> = { q2: false, q3: false, q4: false };
  let otherText: Record<string, string> = { q2: "", q3: "", q4: "" };

  onMount(async () => {
    if (idea.trim().length >= 20) {
      try {
        const dynamic = await fetchDiscoveryHints(idea.trim());
        hints = mergeHints(dynamic);
      } catch {
        hints = mergeHints({});
      }
    }
  });

  $: answeredCount = [
    skipped["1"] || (answers.q1 ?? "").trim().length > 0,
    skipped["2"] || (answers.q2?.length ?? 0) > 0,
    skipped["3"] || (answers.q3?.length ?? 0) > 0,
    skipped["4"] || (answers.q4?.length ?? 0) > 0,
  ].filter(Boolean).length;

  function toggleChip(q: "q2" | "q3" | "q4", label: string) {
    if (label === "+ Other") {
      showOther[q] = !showOther[q];
      return;
    }
    const list = [...(answers[q] ?? [])];
    const idx = list.indexOf(label);
    if (idx >= 0) list.splice(idx, 1);
    else list.push(label);
    answers = { ...answers, [q]: list };
    skipped = { ...skipped, [q.replace("q", "")]: false };
  }

  function skipQuestion(id: string) {
    skipped = { ...skipped, [id]: true };
    if (id === "1") answers = { ...answers, q1: "" };
    if (id === "2") answers = { ...answers, q2: [] };
    if (id === "3") answers = { ...answers, q3: [] };
    if (id === "4") answers = { ...answers, q4: [] };
  }

  function skipAll() {
    dispatch("submit", { q1: "", q2: [], q3: [], q4: [] });
  }

  function submit() {
    const payload: DiscoveryAnswers = {
      q1: skipped["1"] ? "" : (answers.q1 ?? "").trim(),
      q2: skipped["2"] ? [] : [...(answers.q2 ?? [])],
      q3: skipped["3"] ? [] : [...(answers.q3 ?? [])],
      q4: skipped["4"] ? [] : [...(answers.q4 ?? [])],
    };
    for (const q of ["q2", "q3", "q4"] as const) {
      const other = otherText[q].trim();
      const skipKey = q.replace("q", "");
      const current = payload[q] ?? [];
      if (!skipped[skipKey] && other && !current.includes(other)) {
        payload[q] = [...current, other];
      }
    }
    dispatch("submit", payload);
  }
</script>

<div class="clarify-panel">
  <div class="clarify-head">
    <div>
      <h2>Clarify your idea</h2>
      <p class="muted">
        Help agents understand context before they start. All questions are optional.
      </p>
    </div>
    <div class="progress-wrap">
      <div class="cq-counter">{answeredCount} / 4 answered</div>
      <div class="cq-bar-wrap">
        <div class="cq-bar-fill" style="width:{(answeredCount / 4) * 100}%"></div>
      </div>
    </div>
  </div>

  <div class="cq-block">
    <div class="cq-header">
      <div class="cq-num">1</div>
      <div class="cq-question">Who needs this, and what are they doing <em>today</em>?</div>
      <button type="button" class="cq-skip" on:click={() => skipQuestion("1")}>Skip</button>
    </div>
    <textarea
      class="cq-textarea"
      placeholder="e.g. Backend engineers at mid-size SaaS companies…"
      bind:value={answers.q1}
      disabled={skipped["1"]}
    ></textarea>
  </div>

  {#each [
    { id: "2", q: "q2" as const, question: "What must work before you ship to the first real user?", chips: hints.q2 },
    { id: "3", q: "q3" as const, question: "Which of these are non-negotiable?", chips: hints.q3 },
    { id: "4", q: "q4" as const, question: "Why is this better than what users do today?", chips: hints.q4 },
  ] as block (block.id)}
    <div class="cq-divider"></div>
    <div class="cq-block">
      <div class="cq-header">
        <div class="cq-num">{block.id}</div>
        <div class="cq-question">{block.question} <span class="cq-multi-hint">(pick all that apply)</span></div>
        <button type="button" class="cq-skip" on:click={() => skipQuestion(block.id)}>Skip</button>
      </div>
      <div class="cq-chip-grid">
        {#each block.chips as chip}
          <button
            type="button"
            class="q-chip"
            class:selected={(answers[block.q] ?? []).includes(chip)}
            disabled={skipped[block.id]}
            on:click={() => toggleChip(block.q, chip)}
          >{chip}</button>
        {/each}
        <button
          type="button"
          class="q-chip q-chip-other"
          on:click={() => toggleChip(block.q, "+ Other")}
        >+ Other</button>
      </div>
      {#if showOther[block.q]}
        <input
          class="cq-other-input"
          placeholder="Describe…"
          bind:value={otherText[block.q]}
          disabled={skipped[block.id]}
        />
      {/if}
    </div>
  {/each}

  <div class="btn-row">
    <button type="button" class="btn-soft" on:click={() => dispatch("back")}>← Back</button>
    <div class="btn-group">
      <button type="button" class="btn-soft" on:click={skipAll}>Skip all →</button>
      <button type="button" class="btn-primary" on:click={submit}>Start Session</button>
    </div>
  </div>
</div>

<style>
  .clarify-panel { display: grid; gap: 0; }
  .clarify-head {
    display: flex;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 20px;
  }
  h2 { font-size: 1.4rem; margin: 0 0 6px; }
  .muted { color: var(--ink-500); font-size: 0.875rem; margin: 0; }
  .progress-wrap { text-align: right; flex-shrink: 0; }
  .cq-counter { font-size: 0.75rem; font-weight: 600; color: var(--ink-500); }
  .cq-bar-wrap {
    width: 120px;
    height: 6px;
    background: #e2e8f5;
    border-radius: 99px;
    margin-top: 6px;
    overflow: hidden;
  }
  .cq-bar-fill {
    height: 100%;
    background: linear-gradient(90deg, var(--accent-2), var(--accent));
    transition: width 0.2s ease;
  }
  .cq-block { padding: 16px 0; }
  .cq-header { display: flex; gap: 10px; align-items: flex-start; margin-bottom: 10px; }
  .cq-num {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: #e8f0ff;
    color: var(--accent-2);
    font-weight: 700;
    font-size: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .cq-question { flex: 1; font-weight: 600; font-size: 0.875rem; }
  .cq-multi-hint { font-weight: 400; color: var(--ink-500); font-size: 0.75rem; }
  .cq-skip {
    border: none;
    background: transparent;
    color: var(--ink-500);
    font-size: 0.75rem;
    cursor: pointer;
    text-decoration: underline;
  }
  .cq-textarea {
    width: 100%;
    min-height: 88px;
    border: 1px solid #cfd8ea;
    border-radius: 12px;
    padding: 11px 12px;
    font: inherit;
    resize: none;
  }
  .cq-divider { border-top: 1px solid var(--line); }
  .cq-chip-grid { display: flex; flex-wrap: wrap; gap: 8px; }
  .q-chip {
    border: 1px solid #d5dcec;
    background: #fff;
    border-radius: 999px;
    padding: 6px 12px;
    font-size: 0.8125rem;
    cursor: pointer;
  }
  .q-chip.selected {
    border-color: var(--accent-2);
    background: rgba(31, 122, 224, 0.1);
    color: var(--accent-2);
  }
  .q-chip-other { border-style: dashed; }
  .cq-other-input {
    width: 100%;
    margin-top: 8px;
    border: 1px solid #cfd8ea;
    border-radius: 10px;
    padding: 9px 11px;
    font: inherit;
  }
  .btn-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 24px;
    padding-top: 20px;
    border-top: 1px solid var(--line);
  }
  .btn-group { display: flex; gap: 10px; }
  .btn-soft {
    border: 1.5px solid var(--line);
    background: transparent;
    border-radius: 11px;
    padding: 11px 14px;
    font-weight: 600;
    cursor: pointer;
  }
</style>
