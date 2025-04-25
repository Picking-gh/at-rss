<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from "svelte";

  export let showModal: boolean = false;
  export let title: string = "Modal Title";

  const dispatch = createEventDispatcher();

  function closeModal() {
    dispatch("close");
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
  <div class="modal-backdrop" on:click={closeModal} role="presentation">
    <div class="modal-content" role="dialog" aria-modal="true" aria-labelledby="modal-title" on:click|stopPropagation>
      <header class="modal-header">
        <h2 id="modal-title">{title}</h2>
        <button class="close-button" on:click={closeModal} aria-label="Close modal">&times;</button>
      </header>
      <section class="modal-body">
        <slot name="body"></slot>
        <!-- Content goes here -->
      </section>
      <footer class="modal-footer">
        <slot name="footer">
          <!-- Default footer button if needed -->
          <!-- <button class="button" on:click={closeModal}>Close</button> -->
        </slot>
      </footer>
    </div>
  </div>
{/if}
