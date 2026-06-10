<script lang="ts">
  import type { TechConstraints } from "$lib/types";

  export let value: TechConstraints = {
    agents_decide: true,
    must_use: [],
    comfortable_with: [],
    avoid_if_possible: [],
  };

  const mustHints = [
    { cat: "Frontend", chips: ["React", "Next.js", "Vue", "SvelteKit", "Angular"] },
    { cat: "Backend", chips: ["Go", "Node.js", "Java", "Python", "NestJS", "Hono"] },
    { cat: "Database", chips: ["PostgreSQL", "MySQL", "MongoDB", "Redis", "SQLite"] },
    { cat: "Cloud", chips: ["AWS", "GCP", "Azure", "Docker", "Kubernetes"] },
  ];
  const prefHints = [
    { cat: "Frontend", chips: ["React", "Next.js", "Vue", "SvelteKit"] },
    { cat: "Backend", chips: ["Go", "Node.js", "Python", "NestJS"] },
    { cat: "Database", chips: ["PostgreSQL", "Redis", "SQLite"] },
    { cat: "Cloud", chips: ["Docker", "AWS", "GCP"] },
  ];
  const avoidHints = [
    { cat: "Architecture", chips: ["Microservices", "Event sourcing", "GraphQL", "gRPC"] },
    { cat: "Infra", chips: ["Kubernetes", "Serverless", "Terraform", "Helm"] },
    { cat: "Vendor", chips: ["AWS lock-in", "GCP lock-in", "Azure lock-in", "Firebase"] },
    { cat: "Complexity", chips: ["Heavy DevOps", "Manual infra setup", "Complex CI/CD"] },
  ];

  function toggleAgentsDecide() {
    value = { ...value, agents_decide: !value.agents_decide };
  }

  function addChip(field: keyof TechConstraints, label: string) {
    if (field === "agents_decide") return;
    const list = [...((value[field] as string[]) ?? [])];
    if (list.includes(label)) return;
    list.push(label);
    value = { ...value, [field]: list };
  }

  function removeChip(field: keyof TechConstraints, label: string) {
    if (field === "agents_decide") return;
    const list = ((value[field] as string[]) ?? []).filter((x) => x !== label);
    value = { ...value, [field]: list };
  }

  function onInputKey(
    e: KeyboardEvent,
    field: "must_use" | "comfortable_with" | "avoid_if_possible",
  ) {
    const input = e.target as HTMLInputElement;
    if (e.key === "Enter" && input.value.trim()) {
      addChip(field, input.value.trim());
      input.value = "";
      e.preventDefault();
    }
  }
</script>

<div class="constraints-block" class:collapsed={value.agents_decide}>
  <div class="constraints-header">
    <div>
      <div class="field-label">
        Tech Constraints <span class="opt-tag">optional</span>
      </div>
      <p class="muted">Guide agents toward a stack — or let them decide.</p>
    </div>
    <label class="toggle-wrap">
      <input
        type="checkbox"
        checked={value.agents_decide}
        on:change={toggleAgentsDecide}
      />
      <span class="toggle-track"><span class="toggle-knob"></span></span>
      <span class="toggle-label">Let agents decide</span>
    </label>
  </div>

  {#if !value.agents_decide}
    <div class="constraints-body">
      {#each [
        { key: "must_use", label: "Must use", hints: mustHints, placeholder: "Type and press Enter…" },
        { key: "comfortable_with", label: "Comfortable with", hints: prefHints, placeholder: "Preferred technologies…" },
        { key: "avoid_if_possible", label: "Avoid if possible", hints: avoidHints, placeholder: "Technologies to avoid…" },
      ] as tier (tier.key)}
        <div class="tier">
          <div class="tier-label">{tier.label}</div>
          <div class="chip-row">
            {#each value[tier.key as "must_use" | "comfortable_with" | "avoid_if_possible"] ?? [] as chip}
              <button type="button" class="selected-chip" on:click={() => removeChip(tier.key as "must_use" | "comfortable_with" | "avoid_if_possible", chip)}>
                {chip} ×
              </button>
            {/each}
          </div>
          <input
            class="tier-input"
            placeholder={tier.placeholder}
            on:keydown={(e) => onInputKey(e, tier.key as "must_use" | "comfortable_with" | "avoid_if_possible")}
          />
          <div class="hint-block">
            {#each tier.hints as row}
              <div class="hint-row">
                <span class="hint-cat">{row.cat}</span>
                {#each row.chips as chip}
                  <button
                    type="button"
                    class="hint-chip"
                    on:click={() => addChip(tier.key as "must_use" | "comfortable_with" | "avoid_if_possible", chip)}
                  >{chip}</button>
                {/each}
              </div>
            {/each}
          </div>
        </div>
      {/each}
      <div class="constraints-note">
        Constraints are injected into canonical state as <strong>assumptions[]</strong> — agents may challenge them during iteration.
      </div>
    </div>
  {:else}
    <div class="agents-decide-msg">Agents will propose and validate the tech stack during iteration.</div>
  {/if}
</div>

<style>
  .constraints-block {
    border: 1px solid #dce5f5;
    border-radius: 14px;
    background: #f8faff;
    overflow: hidden;
    margin-bottom: 18px;
  }
  .constraints-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding: 14px 16px;
  }
  .opt-tag {
    font-weight: 400;
    font-size: 11px;
    color: var(--ink-500);
    background: #eef1f8;
    border: 1px solid #d5dcec;
    border-radius: 99px;
    padding: 1px 7px;
    margin-left: 6px;
  }
  .toggle-wrap {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    user-select: none;
    flex-shrink: 0;
  }
  .toggle-wrap input {
    position: absolute;
    opacity: 0;
    width: 1px;
    height: 1px;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
  .toggle-track {
    position: relative;
    width: 40px;
    height: 22px;
    border-radius: 999px;
    background: #d5dcec;
    transition: background 0.22s ease;
  }
  .toggle-wrap input:checked + .toggle-track {
    background: linear-gradient(135deg, var(--accent-2), var(--accent));
  }
  .toggle-knob {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 1px 4px rgba(0, 0, 0, 0.18);
    transition: transform 0.22s ease;
  }
  .toggle-wrap input:checked + .toggle-track .toggle-knob {
    transform: translateX(18px);
  }
  .toggle-label { font-size: 0.8125rem; font-weight: 600; color: var(--ink-700); }
  .constraints-body { padding: 0 16px 16px; display: grid; gap: 14px; }
  .tier-label { font-weight: 600; font-size: 0.8125rem; margin-bottom: 6px; }
  .tier-input {
    width: 100%;
    border: 1px solid #cfd8ea;
    border-radius: 10px;
    padding: 9px 11px;
    font: inherit;
    margin: 8px 0;
  }
  .chip-row { display: flex; flex-wrap: wrap; gap: 6px; }
  .selected-chip {
    border: 1px solid #86a4da;
    background: #ddebff;
    border-radius: 999px;
    padding: 4px 10px;
    font-size: 12px;
    cursor: pointer;
  }
  .hint-block { display: grid; gap: 6px; }
  .hint-row { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
  .hint-cat {
    font-size: 11px;
    font-weight: 600;
    color: var(--ink-500);
    min-width: 72px;
  }
  .hint-chip {
    border: 1px solid #cdebf1;
    background: #eefbff;
    color: #0f829d;
    border-radius: 999px;
    padding: 4px 10px;
    font-size: 12px;
    cursor: pointer;
  }
  .hint-chip:hover { background: #dff5fc; }
  .constraints-note {
    margin-top: 4px;
    padding: 9px 11px;
    background: #fffbf0;
    border: 1px solid #f5e0a0;
    border-radius: 9px;
    font-size: 12px;
    color: #7a5700;
  }
  .agents-decide-msg {
    padding: 0 16px 14px;
    font-size: 0.8125rem;
    color: var(--ink-500);
  }
  .muted { color: var(--ink-500); font-size: 0.8125rem; margin: 4px 0 0; }
  .field-label { font-weight: 600; font-size: 0.8125rem; }
</style>
