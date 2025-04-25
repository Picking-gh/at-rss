<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import Modal from "./Modal.svelte"; // Import Modal component

  export let feeds: string[] = []; // Array of feed URLs

  const dispatch = createEventDispatcher();

  // --- SVG Icons ---
  const EDIT_ICON_SVG = `
  <svg width='16px' height='16px' viewBox='0 0 24 24' fill='none' xmlns='http://www.w3.org/2000/svg'>
      <path d='M20,16v4a2,2,0,0,1-2,2H4a2,2,0,0,1-2-2V6A2,2,0,0,1,4,4H8' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' stroke-width='2'/>
      <polygon points='12.5 15.8 22 6.2 17.8 2 8.3 11.5 8 16 12.5 15.8' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' stroke-width='2'/>
  </svg>`;

  const DELETE_ICON_SVG = `
  <svg width='16px' height='16px' viewBox='0 0 24 24' fill='none' xmlns='http://www.w3.org/2000/svg'>
      <path d='M10 12V17 M14 12V17 M4 7H20 M6 10V18C6 19.66 7.34 21 9 21H15C16.66 21 18 19.66 18 18V10 M9 5C9 3.9 9.9 3 11 3H13C14.1 3 15 3.9 15 5V7H9V5Z' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'/>
  </svg>`;

  // --- Modal State ---
  let showFeedModal = false;
  let modalTitle = "";
  let currentFeedValue = "";
  let editingFeedIndex: number | null = null;

  // --- Drag and Drop State ---
  let dragStartIndex: number | null = null;
  let dragOverIndex: number | null = null;

  // --- Event Handlers ---
  function openAddModal() {
    modalTitle = "Add New Feed";
    currentFeedValue = "";
    editingFeedIndex = null;
    showFeedModal = true;
  }

  function openEditModal(index: number) {
    modalTitle = "Edit Feed";
    currentFeedValue = feeds[index];
    editingFeedIndex = index;
    showFeedModal = true;
  }

  function saveFeed() {
    const feedValue = currentFeedValue.trim();
    if (!feedValue) {
      alert("Feed URL cannot be empty.");
      return;
    }

    let updatedFeeds: string[];
    if (editingFeedIndex !== null) {
      // Editing existing feed
      if (feedValue === feeds[editingFeedIndex]) {
        showFeedModal = false; // No change, just close modal
        return;
      }
      updatedFeeds = [...feeds];
      updatedFeeds[editingFeedIndex] = feedValue;
    } else {
      // Adding new feed - check for duplicates
      if (feeds.includes(feedValue)) {
        alert(`Feed URL "${feedValue}" already exists.`);
        return;
      }
      updatedFeeds = [...feeds, feedValue];
    }
    dispatch("update:feeds", updatedFeeds);
    showFeedModal = false; // Close modal after saving
  }

  function handleDelete(index: number) {
    if (confirm(`Are you sure you want to delete feed "${feeds[index]}"?`)) {
      console.log("Delete feed clicked for index:", index);
      const updatedFeeds = feeds.filter((_, i) => i !== index);
      dispatch("update:feeds", updatedFeeds);
    }
  }

  // --- Drag and Drop Handlers ---
  function handleDragStart(event: DragEvent, index: number) {
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = "move";
      event.dataTransfer.setData("text/plain", index.toString());
      dragStartIndex = index;
      (event.target as HTMLLIElement).classList.add("dragging");
    }
  }

  function handleDragOver(event: DragEvent, index: number) {
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "move";
    }
    dragOverIndex = index;
    const targetElement = event.currentTarget as HTMLLIElement;
    if (dragStartIndex !== index) {
      targetElement.classList.add("drag-over");
    }
  }

  function handleDragLeave(event: DragEvent) {
    (event.currentTarget as HTMLLIElement).classList.remove("drag-over");
    dragOverIndex = null;
  }

  function handleDrop(event: DragEvent, dropIndex: number) {
    event.preventDefault();
    (event.currentTarget as HTMLLIElement).classList.remove("drag-over");

    if (dragStartIndex === null || dragStartIndex === dropIndex) {
      dragStartIndex = null;
      return;
    }

    const draggedItem = feeds[dragStartIndex];
    const remainingItems = feeds.filter((_, i) => i !== dragStartIndex);

    const reorderedFeeds = [...remainingItems.slice(0, dropIndex), draggedItem, ...remainingItems.slice(dropIndex)];

    dispatch("update:feeds", reorderedFeeds);
    dragStartIndex = null;
  }

  function handleDragEnd(event: DragEvent) {
    (event.target as HTMLLIElement).classList.remove("dragging");
    document.querySelectorAll(".drag-over").forEach((el) => el.classList.remove("drag-over"));
    dragStartIndex = null;
    dragOverIndex = null;
  }
</script>

<!-- Feed Add/Edit Modal -->
<Modal bind:showModal={showFeedModal} title={modalTitle} on:close={() => (showFeedModal = false)}>
  <div slot="body">
    <div class="form-group">
      <label for="feed-url-input">Feed URL</label>
      <input type="url" id="feed-url-input" bind:value={currentFeedValue} placeholder="https://example.com/rss.xml" required class="modal-input" />
    </div>
  </div>
  <div slot="footer">
    <button type="button" class="button secondary-button" on:click={() => (showFeedModal = false)}>Cancel</button>
    <button type="button" class="button primary-button" on:click={saveFeed}>Save Feed</button>
  </div>
</Modal>

<div class="list-section">
  <h4>Feeds</h4>
  {#if feeds && feeds.length > 0}
    <ul class="list-items" id="feed-list">
      {#each feeds as feed, index (index)}
        <li
          class="list-item draggable-item"
          data-index={index}
          draggable="true"
          on:dragstart={(e) => handleDragStart(e, index)}
          on:dragover={(e) => handleDragOver(e, index)}
          on:dragleave={handleDragLeave}
          on:drop={(e) => handleDrop(e, index)}
          on:dragend={handleDragEnd}
          class:drag-over={dragOverIndex === index && dragStartIndex !== index}
        >
          <span class="item-content">
            <span class="drag-handle">::</span>
            {feed}
          </span>
          <div class="list-item-actions">
            <button type="button" class="button icon-button secondary-button" on:click={() => openEditModal(index)} title="Edit Feed">
              {@html EDIT_ICON_SVG}
            </button>
            <button type="button" class="button icon-button danger-button" on:click={() => handleDelete(index)} title="Delete Feed">
              {@html DELETE_ICON_SVG}
            </button>
          </div>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="empty-list-message">No feeds configured.</p>
  {/if}

  <button type="button" class="button secondary-button add-item-button" on:click={openAddModal}> Add Feed </button>
</div>
