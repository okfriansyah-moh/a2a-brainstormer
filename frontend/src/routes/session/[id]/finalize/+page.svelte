<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/stores";
  import { goto } from "$app/navigation";
  import { sessionStore } from "$lib/stores/sessionStore";
  import { getSession, finalizeSession } from "$lib/services/api";
  import type { Session } from "$lib/types";

  // ── Route param ─────────────────────────────────────────────────────────
  $: sessionId = $page.params.id;

  // ── Component state ─────────────────────────────────────────────────────
  let session: Session | null = null;
  let pageLoading = true;
  let error = "";
  let generating = false;
  let generated = false;
  let alreadyFinalized = false;

  let archMarkdown = "";
  let roadmapMarkdown = "";
  let archStatus: "pending" | "generating" | "done" = "pending";
  let roadStatus: "pending" | "generating" | "done" = "pending";

  let logLines: string[] = [];
  let runningLine: string | null = null;
  let logDone = false;
  let logBadgeDone = false;

  let copiedArch = false;
  let copiedRoadmap = false;

  // ── Log animation sequence ───────────────────────────────────────────────
  const LOG_SEQUENCE = [
    "Reading canonical state snapshot…",
    "Extracting architecture decisions and component boundaries…",
    "Extracting execution plan — steps, milestones, rollback gates…",
    "Extracting risks, assumptions, and open questions…",
    "Assembling architecture.md sections…",
    "Assembling roadmap.md phases and milestones…",
    "Writing output artifacts…",
    "Generation complete. 2 documents ready. ✓",
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
    if (!sid || generating || generated) return;
    error = "";
    generating = true;
    archStatus = "generating";
    roadStatus = "generating";

    try {
      // Run animation and API call in parallel; show content when both finish
      const [resp] = await Promise.all([
        finalizeSession(sid),
        runLogAnimation(),
      ]);

      archMarkdown = resp.architecture_markdown;
      roadmapMarkdown = resp.roadmap_markdown;
      archStatus = "done";
      roadStatus = "done";
      generated = true;
    } catch (err) {
      error = err instanceof Error ? err.message : "Generation failed.";
      archStatus = "pending";
      roadStatus = "pending";
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
    if (archMarkdown) downloadFile(archMarkdown, "architecture.md");
    if (roadmapMarkdown) downloadFile(roadmapMarkdown, "roadmap.md");
  }

  async function copyToClipboard(text: string, which: "arch" | "roadmap") {
    try {
      await navigator.clipboard.writeText(text);
      if (which === "arch") {
        copiedArch = true;
        setTimeout(() => (copiedArch = false), 2000);
      } else {
        copiedRoadmap = true;
        setTimeout(() => (copiedRoadmap = false), 2000);
      }
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

    // Use session store if already loaded for this session
    if ($sessionStore.session_id === sid && $sessionStore.idea) {
      session = {
        id: sid,
        idea: $sessionStore.idea,
        status: "active",
        max_iterations: 10,
        current_state: $sessionStore.state,
        created_at: "",
        updated_at: "",
      } as Session;
      pageLoading = false;
    } else {
      try {
        session = await getSession(sid);
        pageLoading = false;
      } catch (err) {
        error = err instanceof Error ? err.message : "Failed to load session.";
        pageLoading = false;
        return;
      }
    }

    // If session is already approved, auto-load content without animation
    if (session?.status === "approved") {
      alreadyFinalized = true;
      archStatus = "generating";
      roadStatus = "generating";
      try {
        const resp = await finalizeSession(sid);
        archMarkdown = resp.architecture_markdown;
        roadmapMarkdown = resp.roadmap_markdown;
        archStatus = "done";
        roadStatus = "done";
        generated = true;
        logBadgeDone = true;
        logLines = ["Loaded from previously generated session. ✓"];
      } catch {
        // Fallback: let user click "Generate Documents"
        alreadyFinalized = false;
        archStatus = "pending";
        roadStatus = "pending";
      }
    }
  });
</script>

<svelte:head>
  <title>Finalize Session — A2A Brainstorm</title>
</svelte:head>

<div class="artboard">
  <!-- ── Page header ───────────────────────────────────────────────────── -->
  <div class="fin-topbar">
    <div>
      <div class="fin-title">
        {#if session}
          {session.idea}
        {:else if pageLoading}
          Loading session…
        {:else}
          Finalize Session
        {/if}
      </div>
      <div class="fin-subtitle">
        {#if alreadyFinalized}
          Session complete — previously generated documents
        {:else if generated}
          Session complete — 2 documents generated
        {:else if generating}
          Generating output documents…
        {:else}
          Ready to generate output documents
        {/if}
      </div>
    </div>
    <div class="fin-nav">
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
      <a
        href="/history"
        class="topbar-link"
        on:click={(e) => {
          e.preventDefault();
          goto("/history");
        }}>Session History</a
      >
    </div>
  </div>

  {#if error}
    <div class="feedback-error" role="alert">{error}</div>
  {/if}

  {#if pageLoading}
    <p class="loading-msg">Loading session…</p>
  {:else}
    <!-- ── Generate button (shown when not yet generated) ─────────────── -->
    {#if !generated && !alreadyFinalized}
      <div class="fin-cta">
        <button class="btn-primary" disabled={generating} on:click={generate}>
          {#if generating}
            Generating…
          {:else}
            Generate Documents
          {/if}
        </button>
        <p class="fin-cta-hint">
          This will finalize the session and produce <code>architecture.md</code
          >
          and <code>roadmap.md</code>.
        </p>
      </div>
    {/if}

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
        <!-- architecture.md card -->
        <div class="output-card">
          <div class="output-head">
            <div class="output-file">architecture.md</div>
            <span class={statusClass(archStatus)}
              >{statusLabel(archStatus)}</span
            >
          </div>
          <div class="output-desc">
            Component design, data flows, technology choices
          </div>
          <div class="output-preview">
            {#if archMarkdown}
              {archMarkdown}
            {:else}
              Waiting…
            {/if}
          </div>
          <div class="output-actions">
            <button
              class="btn-soft"
              disabled={!archMarkdown}
              on:click={() => copyToClipboard(archMarkdown, "arch")}
            >
              {copiedArch ? "Copied!" : "Copy"}
            </button>
            <button
              class="btn-soft"
              disabled={!archMarkdown}
              on:click={() => downloadFile(archMarkdown, "architecture.md")}
            >
              Download
            </button>
          </div>
        </div>

        <!-- roadmap.md card -->
        <div class="output-card">
          <div class="output-head">
            <div class="output-file">roadmap.md</div>
            <span class={statusClass(roadStatus)}
              >{statusLabel(roadStatus)}</span
            >
          </div>
          <div class="output-desc">
            Phased execution plan with milestones and risks
          </div>
          <div class="output-preview">
            {#if roadmapMarkdown}
              {roadmapMarkdown}
            {:else}
              Waiting…
            {/if}
          </div>
          <div class="output-actions">
            <button
              class="btn-soft"
              disabled={!roadmapMarkdown}
              on:click={() => copyToClipboard(roadmapMarkdown, "roadmap")}
            >
              {copiedRoadmap ? "Copied!" : "Copy"}
            </button>
            <button
              class="btn-soft"
              disabled={!roadmapMarkdown}
              on:click={() => downloadFile(roadmapMarkdown, "roadmap.md")}
            >
              Download
            </button>
          </div>
        </div>
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
        <div class="run-status">Both documents generated successfully.</div>
      </div>
    {/if}
  {/if}
</div>

<style>
  /* ─── Page header ──────────────────────────────────────────────────── */
  .fin-topbar {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    padding: 20px 0 16px;
    gap: 16px;
  }

  .fin-title {
    font-family: "Space Grotesk", sans-serif;
    font-size: 18px;
    font-weight: 700;
    color: var(--ink-900);
    line-height: 1.3;
    max-width: 540px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .fin-subtitle {
    font-size: 13px;
    color: var(--ink-500);
    margin-top: 3px;
  }

  .fin-nav {
    display: flex;
    align-items: center;
    gap: 10px;
    flex-shrink: 0;
  }

  .fin-status-chip {
    font-size: 11px;
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

  .fin-cta-hint code {
    font-family: "IBM Plex Mono", monospace;
    font-size: 12px;
    background: var(--bg-1);
    border: 1px solid var(--line);
    border-radius: 4px;
    padding: 1px 5px;
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
</style>
