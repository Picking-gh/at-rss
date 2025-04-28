<script lang="ts">
  import Modal from "./Modal.svelte";
  import ListItem from "./ListItem.svelte";

  interface Props {
    feeds?: string[]; // Array of feed URLs
    update?: any;
  }

  let { feeds = [], update }: Props = $props();

  // --- Modal State ---
  let showFeedModal = $state(false);
  let modalTitle = $state("");
  let currentFeedValue = $state("");
  let editingFeedIndex: number | null = null;

  // --- Drag and Drop State ---
  let dragStartIndex: number | null = $state(null);
  let dragOverIndex: number | null = $state(null);

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
    update(updatedFeeds);
    showFeedModal = false;
  }

  function handleDelete(index: number) {
    if (confirm(`Are you sure you want to delete feed "${feeds[index]}"?`)) {
      const updatedFeeds = feeds.filter((_, i) => i !== index);
      update(updatedFeeds);
    }
  }

  // --- Drag and Drop Handlers ---
  function handleDragStart(index: number) {
    dragStartIndex = index;
  }

  function handleDragOver(index: number) {
    dragOverIndex = index;
  }

  function handleDragLeave() {
    dragOverIndex = null;
  }

  function handleDrop(index: number) {
    const dropIndex = index;

    if (dragStartIndex === null || dragStartIndex === dropIndex) {
      dragStartIndex = null;
      return;
    }

    const draggedItem = feeds[dragStartIndex];
    const remainingItems = feeds.filter((_, i) => i !== dragStartIndex);
    const reorderedFeeds = [...remainingItems.slice(0, dropIndex), draggedItem, ...remainingItems.slice(dropIndex)];

    update(reorderedFeeds);
    dragStartIndex = null;
  }

  function handleDragEnd() {
    dragStartIndex = null;
    dragOverIndex = null;
  }
</script>

<!-- Feed Add/Edit Modal -->
<Modal bind:showModal={showFeedModal} title={modalTitle} close={() => (showFeedModal = false)}>
  {#snippet body()}
    <div class="form-group">
      <label for="feed-url-input">Feed URL</label>
      <input type="url" id="feed-url-input" bind:value={currentFeedValue} placeholder="https://example.com/rss.xml" required class="modal-input" />
    </div>
  {/snippet}
  {#snippet footer()}
    <button type="button" class="button primary-button" onclick={saveFeed}>Save</button>
    <button type="button" class="button secondary-button" onclick={() => (showFeedModal = false)}>Cancel</button>
  {/snippet}
</Modal>

<div class="form-section">
  <h3>Feeds</h3>
  <div class="list-section">
    {#if feeds && feeds.length > 0}
      <ul class="list-items" id="feed-list">
        {#each feeds as feed, index (index)}
          <ListItem
            item={feed}
            {index}
            draggable={true}
            isDraggedOver={dragOverIndex === index && dragStartIndex !== index}
            dragStart={handleDragStart}
            dragOver={handleDragOver}
            dragLeave={handleDragLeave}
            drop={handleDrop}
            dragEnd={handleDragEnd}
            edit={() => openEditModal(index)}
            del={() => handleDelete(index)}
          >
            {feed}
          </ListItem>
        {/each}
      </ul>
    {:else}
      <p class="empty-list-message">No feeds configured.</p>
    {/if}

    <button type="button" class="button secondary-button add-item-button" onclick={openAddModal}> Add Feed </button>
  </div>
</div>
