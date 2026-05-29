<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import { getSession, finalizeSession } from "$lib/services/api";
  import type { Session, GeneratedDocument } from "$lib/types";

  // ── Route param ─────────────────────────────────────────────────────────
  $: sessionId = $page.params.id;

  // ── Component state ─────────────────────────────────────────────────────
  let session: Session | null = null;
  let pageLoading = true;
  let error = "";
  let generating = false;
  let generated = false;
  let alreadyFinalized = false;

  /** Document key → generated artifact. Populated after generation. */
  let documents: Record<string, GeneratedDocument> = {};
  /** The doc keys the user wants to generate for this finalize call. */
  let selectedDocs: string[] = ["architecture", "roadmap", "plan", "readme"];
  /** All available document types — single source of truth for the picker. */
  const ALL_DOCS = [
    { key: "architecture", label: "Architecture" },
    { key: "roadmap", label: "Roadmap" },
    { key: "plan", label: "Plan" },
    { key: "readme", label: "README" },
  ];
  /** Per-document copied state for clipboard feedback. */
  let copiedDoc: Record<string, boolean> = {};
  /** Overall doc generation status (applied to all docs uniformly). */
  let docStatus: "pending" | "generating" | "done" = "pending";

  // ── Log animation sequence ───────────────────────────────────────────────
  let logLines: string[] = [];
  let runningLine: string | null = null;
  let logDone = false;
  let logBadgeDone = false;

  const LOG_SEQUENCE = [
    "Reading canonical state snapshot…",
    "Extracting architecture decisions and component boundaries…",
    "Extracting execution plan — steps, milestones, rollback gates…",
    "Extracting risks, assumptions, and open questions…",
    "Assembling document sections…",
    "Writing output artifacts…",
    "Generation complete. Documents ready. ✓",
  ];

  function runLogAnimation(): Promise<void> {
    logLines = [];
    runningLine = null;
    logDone = false;
    logBadgeDone = false;

    return new Promise((resolve) => {
      let i = 0;

      function step() {
        if (i >= LOG_SEQUENCE.length) {
          runningLine = null;
          logDone = true;
          logBadgeDone = true;
          resolve();
          return;
        }
        runningLine = LOG_SEQUENCE[i];
        setTimeout(() => {
          logLines = [...logLines, LOG_SEQUENCE[i]];
          runningLine = null;
          i++;
          setTimeout(step, 120);
        }, 400);
      }

      step();
    });
  }

  // ── Generate flow ────────────────────────────────────────────────────────
  async function generate() {
    const sid = $page.params.id;
    if (!sid || generating || selectedDocs.length === 0) return;
    error = "";
    generating = true;
    docStatus = "generating";
    // Allow re-running on an already-finalized session.
    generated = false;

    try {
      // Run animation and API call in parallel; show content when both finish
      const [resp] = await Promise.all([
        finalizeSession(sid, { output_docs: selectedDocs }),
        runLogAnimation(),
      ]);

      documents = resp.documents ?? {};
      docStatus = "done";
      generated = true;
      alreadyFinalized = true;
    } catch (err) {
      error = err instanceof Error ? err.message : "Generation failed.";
      docStatus = "pending";
      logDone = false;
      logBadgeDone = false;
    } finally {
      generating = false;
    }
  }

  // ── File helpers ─────────────────────────────────────────────────────────
  function downloadFile(content: string, filename: string) {
    const blob = new Blob([content], { type: "text/markdown" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  }

  function downloadAll() {
    for (const doc of Object.values(documents)) {
      downloadFile(doc.content, doc.filename);
    }
  }

  async function copyToClipboard(text: string, key: string) {
    try {
      await navigator.clipboard.writeText(text);
      copiedDoc = { ...copiedDoc, [key]: true };
      setTimeout(() => {
        copiedDoc = { ...copiedDoc, [key]: false };
      }, 2000);
    } catch {
      // Clipboard API unavailable — silently ignore
    }
  }

  // ── Status label helpers ─────────────────────────────────────────────────
  function statusClass(s: "pending" | "generating" | "done"): string {
    const map = {
      pending: "os-pending",
      generating: "os-running",
      done: "os-done",
    };
    return `output-status ${map[s]}`;
  }

  function statusLabel(s: "pending" | "generating" | "done"): string {
    const map = {
      pending: "⬖ Pending",
      generating: "⟳ Generating",
      done: "✓ Ready",
    };
    return map[s];
  }

  // ── Mount ────────────────────────────────────────────────────────────────
  onMount(async () => {
    const sid = $page.params.id;
    if (!sid) {
      pageLoading = false;
      return;
    }

    pageLoading = true;
    error = "";

    // Always fetch fresh session status from the API so we get the real
    // status ("converged", "approved", etc.) rather than a stale store value.
    try {
      session = await getSession(sid);
      pageLoading = false;
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to load session.";
      pageLoading = false;
      return;
    }

    if (session?.status === "approved") {
      // Session was previously finalized — reload the markdown without animation.
      // The user can still re-select documents and regenerate from this page.
      alreadyFinalized = true;
      docStatus = "generating";
      // Preserve the previously chosen docs in the picker so the user sees
      // exactly what's loaded; they can tick more boxes and regenerate.
      selectedDocs =
        session.output_docs && session.output_docs.length > 0
          ? session.output_docs
          : ALL_DOCS.map((d) => d.key);
      try {
        const resp = await finalizeSession(sid);
        documents = resp.documents ?? {};
        docStatus = "done";
        generated = true;
        logBadgeDone = true;
        logLines = ["Loaded from previously generated session. ✓"];
      } catch {
        // Fallback: let user click "Generate Documents"
        alreadyFinalized = false;
        docStatus = "pending";
      }
    } else if (session?.status === "converged") {
      // Arrived from the workspace after iterating — DO NOT auto-trigger.
      // The user must explicitly select documents and click Generate so they
      // can choose all four (architecture, roadmap, plan, readme) instead of
      // being locked into the session's stored default.
      selectedDocs =
        session.output_docs && session.output_docs.length > 0
          ? session.output_docs
          : ALL_DOCS.map((d) => d.key);
    } else {
      // Active session: seed selectedDocs from stored session value
      selectedDocs =
        session?.output_docs && session.output_docs.length > 0
          ? session.output_docs
          : ALL_DOCS.map((d) => d.key);
    }
  });
</script>

<svelte:head>
  <title>Finalize Session — A2A Brainstorm</title>
</svelte:head>

<div class="artboard">
  <!-- ── Page header ───────────────────────────────────────────────────── -->
  <div class="topbar session-topbar">
    <div class="fin-head">
      <div class="topbar-title">
        {#if session}
          {session.idea.length > 80
            ? session.idea.slice(0, 77) + "…"
            : session.idea}
        {:else if pageLoading}
          Loading session…
        {:else}
          Finalize Session
        {/if}
      </div>
      <div class="topbar-subtitle">
        {#if alreadyFinalized}
          Session complete — previously generated documents
        {:else if generated}
          Session complete — {Object.keys(documents).length} document{Object.keys(
            documents,
          ).length !== 1
            ? "s"
            : ""} generated
        {:else if generating}
          Generating output documents…
        {:else}
          Ready to generate output documents
        {/if}
      </div>
    </div>
    <div class="fin-topbar-actions">
      {#if alreadyFinalized}
        <span class="chip-ok fin-status-chip">Already finalized</span>
      {/if}
      <a
        href={`/session/${sessionId}`}
        class="topbar-link"
        on:click={(e) => {
          e.preventDefault();
          goto(`/session/${sessionId}`);
        }}>← Back to Session</a
      >
    </div>
  </div>

  <div class="fin-body">
    {#if error}
      <div class="feedback-error" role="alert">{error}</div>
    {/if}

    {#if pageLoading}
      <p class="loading-msg">Loading session…</p>
    {:else}
      <!-- ── Generate / Regenerate selector (always visible) ──────────── -->
      <div class="fin-cta">
        <!-- Docs selector -->
        <div style="margin-bottom:14px;width:100%;">
          <div
            style="display:flex;align-items:center;justify-content:space-between;margin-bottom:7px;"
          >
            <div
              style="font-weight:600;font-size:0.8125rem;color:var(--ink-900);"
            >
              {alreadyFinalized
                ? "Documents — re-select to regenerate with AI"
                : "Documents to Generate"}
            </div>
            <div style="display:flex;gap:10px;">
              <button
                type="button"
                class="btn-soft"
                style="font-size:0.75rem;padding:3px 10px;"
                disabled={generating}
                on:click={() => (selectedDocs = ALL_DOCS.map((d) => d.key))}
                >Select all</button
              >
              <button
                type="button"
                class="btn-soft"
                style="font-size:0.75rem;padding:3px 10px;"
                disabled={generating}
                on:click={() => (selectedDocs = [])}>Clear</button
              >
            </div>
          </div>
          <div style="display:flex;gap:20px;flex-wrap:wrap;">
            {#each ALL_DOCS as doc (doc.key)}
              <label
                style="display:flex;align-items:center;gap:6px;cursor:pointer;font-size:0.875rem;"
              >
                <input
                  type="checkbox"
                  value={doc.key}
                  checked={selectedDocs.includes(doc.key)}
                  disabled={generating}
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
        <button
          class="btn-primary"
          disabled={generating || selectedDocs.length === 0}
          on:click={generate}
        >
          {#if generating}
            {alreadyFinalized ? "Regenerating…" : "Generating…"}
          {:else if alreadyFinalized}
            Regenerate Selected Documents
          {:else}
            Generate Documents
          {/if}
        </button>
        <p class="fin-cta-hint">
          {#if alreadyFinalized}
            Runs the AI agent (with skills) to rewrite each selected document.
            Existing documents stay until the new run completes.
          {:else}
            Finalizes the session and runs the AI agent (with skills) to
            generate the selected documents.
          {/if}
        </p>
      </div>

      <!-- ── Markdown Generator log panel ──────────────────────────────── -->
      {#if generating || generated || alreadyFinalized}
        <div class="panel gen-panel">
          <div class="gen-panel-head">
            <div class="gen-panel-title">Markdown Generator</div>
            <span class="gen-badge {logBadgeDone ? 'gen-done' : 'gen-running'}">
              {logBadgeDone ? "✓ Done" : "⟳ Generating"}
            </span>
          </div>
          <div class="gen-log">
            {#each logLines as line}
              <div class="gen-entry gen-done-line">{line}</div>
            {/each}
            {#if runningLine}
              <div class="gen-entry gen-running-line">
                <span class="dots">{runningLine}</span>
              </div>
            {/if}
            {#if !runningLine && !logDone && !alreadyFinalized && (generating || generated)}
              <div class="gen-entry gen-running-line">
                <span class="dots">Initializing…</span>
              </div>
            {/if}
          </div>
        </div>
      {/if}

      <!-- ── Output file cards ─────────────────────────────────────────── -->
      {#if generating || generated || alreadyFinalized}
        <div class="output-grid">
          {#each Object.entries(documents) as [key, doc] (key)}
            <div class="output-card">
              <div class="output-head">
                <div class="output-file">{doc.filename}</div>
                <span class={statusClass(docStatus)}
                  >{statusLabel(docStatus)}</span
                >
              </div>
              <div class="output-desc">
                {doc.line_count} lines
                {#if doc.source === "ai"}
                  <span class="src-badge src-ai" title="Rewritten by AI"
                    >✦ AI-generated</span
                  >
                {:else if doc.source === "ai_fallback"}
                  <span
                    class="src-badge src-fallback"
                    title="AI pass failed — deterministic scaffold returned"
                    >⚠ AI fallback (template)</span
                  >
                {:else if doc.source === "deterministic"}
                  <span
                    class="src-badge src-det"
                    title="Template-generated (no AI)">⬡ Template</span
                  >
                {/if}
              </div>
              <div class="output-preview">
                {#if doc.content}
                  {doc.content}
                {:else}
                  Waiting…
                {/if}
              </div>
              <div class="output-actions">
                <button
                  class="btn-soft"
                  disabled={!doc.content}
                  on:click={() => copyToClipboard(doc.content, key)}
                >
                  {copiedDoc[key] ? "Copied!" : "Copy"}
                </button>
                <button
                  class="btn-soft"
                  disabled={!doc.content}
                  on:click={() => downloadFile(doc.content, doc.filename)}
                >
                  Download
                </button>
              </div>
            </div>
          {:else}
            {#if generating}
              <div
                class="output-card"
                style="grid-column:1/-1;text-align:center;color:var(--ink-500);"
              >
                Generating documents…
              </div>
            {/if}
          {/each}
        </div>
      {/if}

      <!-- ── Done bar ───────────────────────────────────────────────────── -->
      {#if generated || alreadyFinalized}
        <div class="panel run-bar">
          <div class="run-left">
            <button class="btn-primary" on:click={downloadAll}>
              Download All
            </button>
            <a
              href="/"
              class="btn-ghost"
              on:click={(e) => {
                e.preventDefault();
                goto("/");
              }}>New Session</a
            >
          </div>
          <div class="run-status">
            {Object.keys(documents).length} document{Object.keys(documents)
              .length !== 1
              ? "s"
              : ""} generated successfully.
          </div>
        </div>
      {/if}
    {/if}
  </div>
</div>

<style>
  /* ─── Page header ──────────────────────────────────────────────────── */
  .session-topbar {
    border-radius: 18px 18px 0 0;
    padding: 0 28px;
  }

  .fin-head {
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-width: 0;
    flex: 1;
  }

  .fin-topbar-actions {
    display: flex;
    align-items: center;
    gap: 14px;
    margin-left: auto;
    flex-shrink: 0;
  }

  .fin-status-chip {
    font-size: 11px;
  }

  .fin-body {
    padding: 20px 28px 28px;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  /* ─── CTA ─────────────────────────────────────────────────────────── */
  .fin-cta {
    display: flex;
    align-items: center;
    gap: 14px;
    margin-bottom: 18px;
    flex-wrap: wrap;
  }

  .fin-cta-hint {
    font-size: 13px;
    color: var(--ink-500);
    margin: 0;
  }

  /* ─── Error / loading ─────────────────────────────────────────────── */
  .feedback-error {
    margin-bottom: 14px;
    padding: 10px 14px;
    border-radius: 8px;
    background: rgba(206, 49, 88, 0.06);
    border: 1px solid rgba(206, 49, 88, 0.3);
    color: var(--danger);
    font-size: 13px;
  }

  .loading-msg {
    color: var(--ink-300);
    font-size: 13px;
    padding: 24px 0;
    margin: 0;
  }

  /* ─── Generator log panel ─────────────────────────────────────────── */
  .gen-panel {
    padding: 18px 20px;
    margin-bottom: 14px;
  }

  .gen-panel-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 14px;
  }

  .gen-panel-title {
    font-family: "Space Grotesk", sans-serif;
    font-weight: 600;
    font-size: 15px;
    color: var(--ink-900);
  }

  .gen-badge {
    font-size: 11px;
    font-weight: 600;
    padding: 3px 9px;
    border-radius: 99px;
  }

  .gen-badge.gen-running {
    background: rgba(31, 122, 224, 0.1);
    color: var(--accent-2);
  }

  .gen-badge.gen-done {
    background: rgba(27, 159, 102, 0.12);
    color: var(--ok);
  }

  .gen-log {
    font-family: "IBM Plex Mono", monospace;
    font-size: 12px;
    color: var(--ink-700);
    line-height: 1.85;
    background: var(--bg-1);
    border-radius: 8px;
    padding: 14px 16px;
    max-height: 200px;
    overflow-y: auto;
  }

  .gen-entry {
    padding: 1px 0;
  }

  .gen-running-line {
    color: var(--accent-2);
  }

  .gen-done-line {
    color: var(--ok);
  }

  /* ─── Output cards ────────────────────────────────────────────────── */
  .output-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
    margin-bottom: 14px;
  }

  @media (max-width: 680px) {
    .output-grid {
      grid-template-columns: 1fr;
    }
  }

  .output-card {
    background: var(--surface);
    border: 1.5px solid var(--line);
    border-radius: 12px;
    padding: 18px 20px;
    display: flex;
    flex-direction: column;
  }

  .output-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 4px;
  }

  .output-file {
    font-family: "IBM Plex Mono", monospace;
    font-weight: 600;
    font-size: 14px;
    color: var(--ink-900);
  }

  .output-status {
    font-size: 11px;
    font-weight: 600;
    padding: 3px 9px;
    border-radius: 99px;
  }

  .os-pending {
    background: var(--bg-1);
    color: var(--ink-500);
  }

  .os-running {
    background: rgba(31, 122, 224, 0.1);
    color: var(--accent-2);
  }

  .os-done {
    background: rgba(27, 159, 102, 0.12);
    color: var(--ok);
  }

  .output-desc {
    font-size: 12px;
    color: var(--ink-500);
    margin-bottom: 12px;
  }

  .output-preview {
    font-family: "IBM Plex Mono", monospace;
    font-size: 11px;
    color: var(--ink-700);
    background: var(--bg-1);
    border-radius: 8px;
    padding: 12px 14px;
    min-height: 140px;
    white-space: pre-wrap;
    word-break: break-word;
    line-height: 1.75;
    flex: 1;
    overflow-y: auto;
    max-height: 320px;
  }

  .output-actions {
    display: flex;
    gap: 8px;
    margin-top: 14px;
  }

  .btn-soft {
    display: inline-block;
    border: 1.5px solid var(--line);
    background: transparent;
    border-radius: 8px;
    padding: 6px 14px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    color: var(--ink-700);
    font-family: "IBM Plex Sans", sans-serif;
    transition: background 0.12s;
  }

  .btn-soft:hover:not(:disabled) {
    background: var(--bg-1);
  }

  .btn-soft:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  /* ─── Done / run bar ──────────────────────────────────────────────── */
  .run-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 18px;
    gap: 12px;
    flex-wrap: wrap;
    margin-top: 4px;
  }

  .run-left {
    display: flex;
    gap: 10px;
    align-items: center;
  }

  .run-status {
    font-size: 13px;
    font-weight: 500;
    color: var(--ok);
  }

  /* ─── Dots animation ─────────────────────────────────────────────── */
  .dots::after {
    content: "";
    display: inline-block;
    animation: blink 1s steps(3, end) infinite;
  }

  @keyframes blink {
    0% {
      content: "";
    }
    33% {
      content: ".";
    }
    66% {
      content: "..";
    }
    100% {
      content: "...";
    }
  }

  /* ─── Source provenance badge ─────────────────────────────────────── */
  .src-badge {
    display: inline-block;
    margin-left: 8px;
    font-size: 10.5px;
    font-weight: 700;
    padding: 2px 8px;
    border-radius: 99px;
    letter-spacing: 0.02em;
    vertical-align: middle;
  }

  .src-ai {
    background: rgba(132, 78, 222, 0.12);
    color: #5b3aa8;
  }

  .src-fallback {
    background: rgba(220, 154, 24, 0.14);
    color: #8a5b00;
  }

  .src-det {
    background: var(--bg-1);
    color: var(--ink-500);
  }
</style>
