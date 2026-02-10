<script lang="ts">
  import { onDestroy, onMount } from "svelte";

  interface Props {
    showModal?: boolean;
    title?: string;
    close?: () => void;
    body?: import("svelte").Snippet;
    footer?: import("svelte").Snippet;
  }

  let { showModal = $bindable(false), title = "Modal Title", close, body, footer }: Props = $props();

  function closeModal() {
    if (typeof close === "function") {
      close();
    }
     // The parent component should handle closing
  }

  // Handle Escape key press
  function handleKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      closeModal();
    }
  }

  onMount(() => {
    window.addEventListener("keydown", handleKeydown);
  });

  onDestroy(() => {
    window.removeEventListener("keydown", handleKeydown);
  });
</script>

{#if showModal}
  <div class="modal-backdrop" role="presentation">
    <div class="modal-content" role="dialog" aria-modal="true" aria-labelledby="modal-title">
      <header class="modal-header">
        <h2 id="modal-title">{title}</h2>
        <button class="close-button" onclick={closeModal} aria-label="Close modal">&times;</button>
      </header>
      <section class="modal-body">
         {@render body?.()}
      </section>
      <footer class="modal-footer">
         {@render footer?.()}
      </footer>
    </div>
  </div>
{/if}

