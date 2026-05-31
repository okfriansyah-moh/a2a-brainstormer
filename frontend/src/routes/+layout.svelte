<script lang="ts">
  import "../app.css";
  import { goto } from "$app/navigation";
  import WarningModal from "$lib/components/WarningModal.svelte";
  import { uiStore } from "$lib/stores/uiStore";
  import { page } from "$app/stores";

  function handleNavClick(event: MouseEvent, href: string): void {
    // Keep native anchor fallback while forcing client-side navigation when hydrated.
    event.preventDefault();
    void goto(href);
  }
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
  <a href="/" class="topbar-logo" on:click={(e) => handleNavClick(e, "/")}
    >A2A Brainstorm</a
  >
  <nav class="topbar-nav">
    {#if String($page.url.pathname) === "/history"}
      <a href="/" class="topbar-link" on:click={(e) => handleNavClick(e, "/")}
        >New Session</a
      >
    {:else}
      <a
        href="/history"
        class="topbar-link"
        class:active={String($page.url.pathname) === "/history"}
        on:click={(e) => handleNavClick(e, "/history")}>Session History</a
      >
    {/if}
    <a
      href="/settings"
      class="topbar-link"
      class:active={String($page.url.pathname).startsWith("/settings")}
      on:click={(e) => handleNavClick(e, "/settings")}>⚙ Settings</a
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
