<script lang="ts">
  export let open = false;
  export let title = "";
  export let body = "";
  export let confirmLabel = "Confirm";
  export let confirmDanger = false;
  export let onConfirm: () => void = () => {};
  export let onDismiss: () => void = () => {};

  let dismissBtn: HTMLButtonElement;
  let confirmBtn: HTMLButtonElement;

  $: if (open) {
    // Move focus into modal after DOM update
    setTimeout(() => confirmBtn?.focus(), 30);
  }

  function handleKeydown(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === "Escape") {
      e.preventDefault();
      onDismiss();
      return;
    }
    if (e.key === "Tab") {
      const focusable = [dismissBtn, confirmBtn].filter(Boolean);
      if (focusable.length < 2) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
  <div class="modal-overlay-wrapper">
    <button
      class="modal-backdrop"
      type="button"
      aria-label="Close dialog"
      tabindex="-1"
      on:click={onDismiss}
    ></button>
    <div
      class="modal"
      role="dialog"
      aria-modal="true"
      aria-labelledby="warn-modal-title"
      tabindex="-1"
    >
      <div class="modal-title" id="warn-modal-title">{title}</div>
      <div class="modal-body">{body}</div>
      <div class="modal-actions">
        <button
          class="btn-ghost"
          type="button"
          bind:this={dismissBtn}
          on:click={onDismiss}
        >
          Cancel
        </button>
        <button
          class={confirmDanger ? "btn-danger" : "btn-primary"}
          type="button"
          bind:this={confirmBtn}
          on:click={onConfirm}
        >
          {confirmLabel}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .modal-overlay-wrapper {
    position: fixed;
    inset: 0;
    z-index: 999;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .modal-backdrop {
    position: absolute;
    inset: 0;
    background: rgba(10, 14, 26, 0.52);
    backdrop-filter: blur(4px);
    border: none;
    cursor: default;
    padding: 0;
  }

  .modal {
    position: relative;
    z-index: 1;
    background: var(--bg-0);
    border: 1.5px solid var(--line);
    border-radius: 14px;
    padding: 28px 28px 24px;
    max-width: 400px;
    width: 90%;
    box-shadow: var(--shadow-md);
  }

  .modal-title {
    font-family: "Space Grotesk", sans-serif;
    font-size: 17px;
    font-weight: 700;
    color: var(--ink-900);
    margin-bottom: 8px;
  }

  .modal-body {
    font-size: 14px;
    color: var(--ink-500);
    line-height: 1.55;
    margin-bottom: 22px;
  }

  .modal-actions {
    display: flex;
    gap: 10px;
    justify-content: flex-end;
  }
</style>
