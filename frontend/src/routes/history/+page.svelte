<script lang="ts">
  import { onMount } from "svelte";
  import { listSessions } from "$lib/services/api";
  import type { SessionListItem } from "$lib/types";

  let sessions: SessionListItem[] = [];
  let loading = true;
  let error = "";
  let searchQuery = "";

  // ── Filtering ─────────────────────────────────────────────────────────────

  $: filteredSessions = sessions.filter((s) =>
    s.idea.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  // ── Stats (computed from filteredSessions) ────────────────────────────────

  $: completedCount = filteredSessions.filter(
    (s) => s.status === "approved" || s.status === "converged",
  ).length;

  $: avgConfidence =
    filteredSessions.length > 0
      ? Math.round(
          (filteredSessions.reduce((sum, s) => sum + s.confidence, 0) /
            filteredSessions.length) *
            100,
        )
      : 0;

  $: docsGenerated = filteredSessions.filter(
    (s) => s.status === "approved",
  ).length;

  $: avgIterations =
    filteredSessions.length > 0
      ? (
          filteredSessions.reduce((sum, s) => sum + s.current_iteration, 0) /
          filteredSessions.length
        ).toFixed(1)
      : "0";

  // ── Helpers ───────────────────────────────────────────────────────────────

  function formatDate(iso: string): string {
    return new Date(iso).toLocaleDateString("en-US", {
      year: "numeric",
      month: "short",
      day: "numeric",
    });
  }

  function confidencePillClass(confidence: number): string {
    if (confidence >= 0.8) return "conf-pill cp-high";
    if (confidence >= 0.5) return "conf-pill cp-mid";
    return "conf-pill cp-low";
  }

  function statusChipClass(status: string): string {
    const map: Record<string, string> = {
      approved: "chip-ok",
      converged: "chip-ok",
      active: "chip-warn",
      failed: "chip-danger",
    };
    return map[status] ?? "chip-warn";
  }

  function viewHref(item: SessionListItem): string {
    return item.status === "approved"
      ? `/session/${item.id}/finalize`
      : `/session/${item.id}`;
  }

  onMount(async () => {
    loading = true;
    error = "";
    try {
      const resp = await listSessions();
      sessions = resp.sessions;
    } catch (err) {
      error =
        err instanceof Error ? err.message : "Failed to load session history.";
    } finally {
      loading = false;
    }
  });
</script>

<svelte:head>
  <title>Session History — A2A Brainstorm</title>
</svelte:head>

<div class="artboard">
  <!-- ── Page header ───────────────────────────────────────────────────── -->
  <div class="history-header">
    <div>
      <div class="history-title">Session History</div>
      <div class="history-subtitle">
        All brainstorm sessions and their generated outputs
      </div>
    </div>
    <nav class="topbar-nav">
      <a href="/" class="topbar-link">New Session</a>
      <a href="/settings" class="topbar-link">⚙ Settings</a>
    </nav>
  </div>

  {#if error}
    <div class="feedback-error" role="alert">{error}</div>
  {/if}

  <!-- ── Stat cards ─────────────────────────────────────────────────────── -->
  <div class="stat-grid">
    <div class="stat-card">
      <div class="stat-val">{completedCount}</div>
      <div class="stat-label">Sessions completed</div>
    </div>
    <div class="stat-card">
      <div class="stat-val">
        {avgConfidence}<span class="stat-unit">%</span>
      </div>
      <div class="stat-label">Avg. final confidence</div>
    </div>
    <div class="stat-card">
      <div class="stat-val">{docsGenerated}</div>
      <div class="stat-label">Documents generated</div>
    </div>
    <div class="stat-card">
      <div class="stat-val">{avgIterations}</div>
      <div class="stat-label">Avg. iterations per session</div>
    </div>
  </div>

  <!-- ── Sessions panel ────────────────────────────────────────────────── -->
  <div class="panel history-panel">
    <!-- Search -->
    <div class="history-search">
      <input
        type="search"
        placeholder="Search sessions by topic or date…"
        bind:value={searchQuery}
        autocomplete="off"
      />
    </div>

    {#if loading}
      <p class="loading-msg">Loading sessions…</p>
    {:else if sessions.length === 0}
      <!-- Empty state -->
      <div class="empty-state">
        <p>No sessions yet.</p>
        <a
          href="/"
          class="btn-primary"
          style="display:inline-block;text-decoration:none;margin-top:12px;"
        >
          Start your first session
        </a>
      </div>
    {:else if filteredSessions.length === 0}
      <div class="empty-state">
        <p>No sessions match <em>"{searchQuery}"</em>.</p>
      </div>
    {:else}
      <!-- Table -->
      <table>
        <thead>
          <tr>
            <th>Session</th>
            <th>Date</th>
            <th>Iterations</th>
            <th>Confidence</th>
            <th>Agents</th>
            <th>Outputs</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each filteredSessions as item (item.id)}
            <tr>
              <td class="idea-cell">
                <strong>{item.idea}</strong>
              </td>
              <td class="date-cell">{formatDate(item.created_at)}</td>
              <td class="num-cell">{item.current_iteration}</td>
              <td>
                <span class={confidencePillClass(item.confidence)}>
                  {Math.round(item.confidence * 100)}%
                </span>
              </td>
              <td>
                <span class="agent-chip"
                  >{item.agent_count} agent{item.agent_count === 1
                    ? ""
                    : "s"}</span
                >
              </td>
              <td>
                {#if item.status === "approved"}
                  <span class="out-chip">arch.md</span>
                  <span class="out-chip">roadmap.md</span>
                {:else}
                  <span class="dim-label">—</span>
                {/if}
              </td>
              <td>
                <span class={statusChipClass(item.status)}>{item.status}</span>
              </td>
              <td>
                <a class="btn-action" href={viewHref(item)}>View →</a>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>

      {#if searchQuery}
        <p class="filter-note">
          Showing {filteredSessions.length} of {sessions.length} sessions
        </p>
      {/if}
    {/if}
  </div>
</div>

<style>
  /* ─── Page header ───────────────────────────────────────────────── */
  .history-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 20px 0 16px;
  }

  .history-title {
    font-family: "Space Grotesk", sans-serif;
    font-size: 22px;
    font-weight: 700;
    color: var(--ink-900);
    line-height: 1.2;
  }

  .history-subtitle {
    font-size: 13px;
    color: var(--ink-500);
    margin-top: 3px;
  }

  /* ─── Error / feedback ─────────────────────────────────────────── */
  .feedback-error {
    margin-bottom: 14px;
    padding: 10px 14px;
    border-radius: 8px;
    background: rgba(206, 49, 88, 0.06);
    border: 1px solid rgba(206, 49, 88, 0.3);
    color: var(--danger);
    font-size: 13px;
  }

  /* ─── Stat grid ────────────────────────────────────────────────── */
  .stat-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 14px;
    margin-bottom: 20px;
  }

  .stat-card {
    background: var(--bg-1);
    border: 1.5px solid var(--line);
    border-radius: 10px;
    padding: 16px 18px;
  }

  .stat-val {
    font-family: "Space Grotesk", sans-serif;
    font-size: 28px;
    font-weight: 700;
    color: var(--ink-900);
    line-height: 1;
    margin-bottom: 4px;
  }

  .stat-unit {
    font-size: 16px;
    font-weight: 500;
    color: var(--ink-700);
  }

  .stat-label {
    font-size: 12px;
    color: var(--ink-500);
  }

  @media (max-width: 700px) {
    .stat-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  /* ─── Panel ─────────────────────────────────────────────────────── */
  .history-panel {
    padding: 20px 24px;
  }

  /* ─── Search ────────────────────────────────────────────────────── */
  .history-search {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 16px;
  }

  .history-search input {
    flex: 1;
    border: 1.5px solid var(--line);
    border-radius: 8px;
    padding: 8px 12px;
    font-size: 14px;
    background: var(--bg-1);
    color: var(--ink-900);
    outline: none;
    font-family: "IBM Plex Sans", sans-serif;
    transition: border-color 0.15s;
  }

  .history-search input:focus {
    border-color: var(--accent);
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

  .idea-cell {
    max-width: 280px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .date-cell {
    color: var(--ink-500);
    font-size: 13px;
    white-space: nowrap;
  }

  .num-cell {
    color: var(--ink-700);
    font-size: 13px;
    text-align: center;
  }

  /* ─── Confidence pills ─────────────────────────────────────────── */
  .conf-pill {
    display: inline-block;
    font-size: 12px;
    font-weight: 700;
    padding: 2px 10px;
    border-radius: 20px;
  }

  .cp-high {
    background: rgba(27, 159, 102, 0.12);
    color: var(--ok);
  }

  .cp-mid {
    background: rgba(212, 136, 6, 0.12);
    color: var(--warn);
  }

  .cp-low {
    background: rgba(206, 49, 88, 0.12);
    color: var(--danger);
  }

  /* ─── Agent chips ───────────────────────────────────────────────── */
  .agent-chip {
    display: inline-block;
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 20px;
    background: rgba(31, 122, 224, 0.1);
    color: var(--accent-2);
    font-weight: 500;
  }

  /* ─── Output chips ───────────────────────────────────────────────── */
  .out-chip {
    display: inline-block;
    font-size: 11px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 20px;
    background: rgba(27, 159, 102, 0.1);
    color: var(--ok);
    margin-right: 4px;
    font-family: "IBM Plex Mono", monospace;
  }

  /* ─── Action button ──────────────────────────────────────────────── */
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
    font-family: "IBM Plex Sans", sans-serif;
    text-decoration: none;
    transition: background 0.12s;
  }

  .btn-action:hover {
    background: var(--bg-1);
  }

  /* ─── Misc ───────────────────────────────────────────────────────── */
  .dim-label {
    color: var(--ink-300);
    font-size: 13px;
  }

  .filter-note {
    font-size: 12px;
    color: var(--ink-500);
    margin-top: 12px;
    margin-bottom: 0;
  }

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
</style>
