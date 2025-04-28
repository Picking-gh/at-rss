<script lang="ts">
  import { onDestroy, onMount } from "svelte";

  interface Props {
    showModal?: boolean;
    title?: string;
    close?: any;
    body?: import("svelte").Snippet;
    footer?: import("svelte").Snippet;
  }

  let { showModal = $bindable(false), title = "Modal Title", close, body, footer }: Props = $props();

  function closeModal() {
    close();
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
        <!-- Content goes here -->
        {@render body?.()}
      </section>
      <footer class="modal-footer">
        <!-- footer buttons go here -->
        {@render footer?.()}
      </footer>
    </div>
  </div>
{/if}

<style>
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgba(0, 0, 0, 0.6);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000; /* Ensure it's on top */
  }

  .modal-content {
    background-color: white;
    padding: 1.5rem 2rem;
    border-radius: 8px;
    box-shadow: 0 5px 15px rgba(0, 0, 0, 0.3);
    min-width: 400px;
    max-width: 600px;
    max-height: 80vh; /* Limit height */
    display: flex;
    flex-direction: column;
    overflow: hidden; /* Prevent content overflow */
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid #eee;
    padding-bottom: 0.8rem;
    margin-bottom: 1rem;
  }

  .modal-header h2 {
    margin: 0;
    font-size: 1.4em;
  }

  .close-button {
    background: none;
    border: none;
    font-size: 1.8rem;
    line-height: 1;
    cursor: pointer;
    color: #888;
    padding: 0 0.5rem;
  }
  .close-button:hover {
    color: #333;
  }

  .modal-body {
    overflow-y: auto; /* Allow body content to scroll */
    margin-bottom: 1rem;
    padding-left: 2px;
    padding-right: 5px; /* Space for scrollbar */
  }

  .modal-footer {
    border-top: 1px solid #eee;
    padding-top: 1rem;
    display: flex;
    justify-content: flex-end; /* Align buttons to the right */
    gap: 0.5rem; /* Space between footer buttons */
  }

  /* --- Responsive Design --- */
  @media (max-width: 768px) {
    .modal-content {
      min-width: 80%;
    }
  }
</style>
