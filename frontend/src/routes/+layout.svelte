<script lang="ts">
  import "../app.css";
  import WarningModal from "$lib/components/WarningModal.svelte";
  import { uiStore } from "$lib/stores/uiStore";
  import { page } from "$app/stores";
</script>

<svelte:head>
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link
    rel="preconnect"
    href="https://fonts.gstatic.com"
    crossorigin="anonymous"
  />
  <link
    href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400&family=IBM+Plex+Sans:wght@300;400;500&family=Space+Grotesk:wght@500;700&display=swap"
    rel="stylesheet"
  />
</svelte:head>

<div class="topbar">
  <a href="/" class="topbar-logo">A2A Brainstorm</a>
  <nav class="topbar-nav">
    <a
      href="/history"
      class="topbar-link"
      class:active={$page.url.pathname === "/history"}>Session History</a
    >
    <a
      href="/settings"
      class="topbar-link"
      class:active={$page.url.pathname.startsWith("/settings")}>⚙ Settings</a
    >
  </nav>
</div>

<slot />

<WarningModal
  open={$uiStore.modalOpen}
  title={$uiStore.modalTitle}
  body={$uiStore.modalBody}
  confirmLabel={$uiStore.modalConfirmLabel}
  confirmDanger={$uiStore.modalConfirmDanger}
  onConfirm={() => {
    const cb = $uiStore.onModalConfirm;
    uiStore.closeModal();
    cb?.();
  }}
  onDismiss={() => uiStore.closeModal()}
/>
