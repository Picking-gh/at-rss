<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import Modal from "./Modal.svelte";
  import ListItem from "./ListItem.svelte";

  export let feeds: string[] = []; // Array of feed URLs

  const dispatch = createEventDispatcher();

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
  function handleDragStart(event: CustomEvent) {
    dragStartIndex = event.detail.index;
  }

  function handleDragOver(event: CustomEvent) {
    dragOverIndex = event.detail.index;
  }

  function handleDragLeave() {
    dragOverIndex = null;
  }

  function handleDrop(event: CustomEvent) {
    const dropIndex = event.detail.index;
    
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

  function handleDragEnd() {
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
    <button type="button" class="button primary-button" on:click={saveFeed}>Save</button>
  </div>
</Modal>

<div class="form-section">
  <h3>Feeds</h3>
  <div class="list-section">
    {#if feeds && feeds.length > 0}
      <ul class="list-items" id="feed-list">
        {#each feeds as feed, index (index)}
          <ListItem
            item={feed}
            index={index}
            draggable={true}
            bind:dragStartIndex
            bind:dragOverIndex
            on:dragstart={handleDragStart}
            on:dragover={handleDragOver}
            on:dragleave={handleDragLeave}
            on:drop={handleDrop}
            on:dragend={handleDragEnd}
            on:edit={() => openEditModal(index)}
            on:delete={() => handleDelete(index)}
          >
            {feed}
          </ListItem>
        {/each}
      </ul>
    {:else}
      <p class="empty-list-message">No feeds configured.</p>
    {/if}

    <button type="button" class="button secondary-button add-item-button" on:click={openAddModal}> Add Feed </button>
  </div>
</div>
