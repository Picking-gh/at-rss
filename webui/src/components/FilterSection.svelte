<script lang="ts">
  import Modal from "./Modal.svelte";
  import ListItem from "./ListItem.svelte";

  // Define the structure for the filter object
  interface FilterConfig {
    include?: string[];
    exclude?: string[];
  }

  interface Props {
    filter?: FilterConfig | null | undefined; // The filter configuration object
    update?: any;
  }

  let { filter = null, update }: Props = $props();

  // --- Local State ---
  // Use derived state or direct binding if possible, but for complex objects, local copy might be easier
  let internalFilter: FilterConfig = $state(filter ? filter : { include: [], exclude: [] });

  // --- Modal State ---
  let showKeywordModal = $state(false);
  let modalTitle = $state("");
  let currentKeywordValue = $state("");
  let editingKeywordType: "include" | "exclude" | null = null;
  let editingKeywordIndex: number | null = null;

  // --- Event Handlers ---
  function notifyUpdate() {
    // Ensure include/exclude are always arrays, even if empty
    const updatedFilter: FilterConfig = {
      include: internalFilter.include || [],
      exclude: internalFilter.exclude || [],
    };
    update(updatedFilter);
  }

  function openAddKeywordModal(type: "include" | "exclude") {
    modalTitle = `Add ${type.charAt(0).toUpperCase() + type.slice(1)} Keyword`;
    currentKeywordValue = "";
    editingKeywordType = type;
    editingKeywordIndex = null;
    showKeywordModal = true;
  }

  function openEditKeywordModal(type: "include" | "exclude", index: number) {
    const currentKeyword = internalFilter[type]?.[index];
    if (currentKeyword === undefined) return;

    modalTitle = `Edit ${type.charAt(0).toUpperCase() + type.slice(1)} Keyword`;
    currentKeywordValue = currentKeyword;
    editingKeywordType = type;
    editingKeywordIndex = index;
    showKeywordModal = true;
  }

  function saveKeyword() {
    const keywordValue = currentKeywordValue.trim();
    if (!keywordValue || !editingKeywordType) {
      alert("Keyword cannot be empty.");
      return;
    }

    const list = internalFilter[editingKeywordType] || [];

    if (editingKeywordIndex !== null) {
      // Editing existing keyword
      const originalKeyword = list[editingKeywordIndex];
      if (keywordValue === originalKeyword) {
        showKeywordModal = false; // No change
        return;
      }
      // Check if edited keyword already exists elsewhere in the list
      if (list.some((kw, i) => i !== editingKeywordIndex && kw === keywordValue)) {
        alert(`Keyword "${keywordValue}" already exists in ${editingKeywordType} list.`);
        return;
      }
      const updatedKeywords = [...list];
      updatedKeywords[editingKeywordIndex] = keywordValue;
      internalFilter[editingKeywordType] = updatedKeywords;
    } else {
      // Adding new keyword - check for duplicates
      if (list.includes(keywordValue)) {
        alert(`Keyword "${keywordValue}" already exists in ${editingKeywordType} list.`);
        return;
      }
      internalFilter[editingKeywordType] = [...list, keywordValue];
    }

    notifyUpdate();
    showKeywordModal = false;
  }

  function deleteKeyword(type: "include" | "exclude", index: number) {
    if (confirm(`Are you sure you want to delete this ${type} keyword?`)) {
      internalFilter[type] = internalFilter[type]?.filter((_, i) => i !== index);
      notifyUpdate();
    }
  }

  function addFilterSection() {
    update({ include: [], exclude: [] });
  }

  function removeFilterSection() {
    if (confirm("Are you sure you want to remove the entire filter section?")) {
      update(null);
    }
  }
</script>

<!-- Keyword Add/Edit Modal -->
<Modal bind:showModal={showKeywordModal} title={modalTitle} close={() => (showKeywordModal = false)}>
  {#snippet body()}
    <div class="form-group">
      <label for="keyword-input">Keyword</label>
      <input type="text" id="keyword-input" bind:value={currentKeywordValue} placeholder="Enter keyword" required class="modal-input" />
    </div>
  {/snippet}
  {#snippet footer()}
    <button type="button" class="button primary-button" onclick={saveKeyword}>Save</button>
    <button type="button" class="button secondary-button" onclick={() => (showKeywordModal = false)}>Cancel</button>
  {/snippet}
</Modal>

<div class="form-section">
  <h3>Filter</h3>
  {#if filter === null || filter === undefined}
    <button type="button" class="button secondary-button" onclick={addFilterSection}> Add Filter Section </button>
  {:else}
    <!-- Include Keywords List -->
    <div class="form-subsection">
      <h4>Include Keywords</h4>
      {#if internalFilter.include && internalFilter.include.length > 0}
        <ul class="list-items keyword-list">
          {#each internalFilter.include as keyword, index (index)}
            <ListItem item={keyword} {index} draggable={false} edit={() => openEditKeywordModal("include", index)} del={() => deleteKeyword("include", index)}>
              {keyword}
            </ListItem>
          {/each}
        </ul>
      {:else}
        <p class="empty-list-message">No include keywords.</p>
      {/if}
      <button type="button" class="button secondary-button add-item-button" onclick={() => openAddKeywordModal("include")}> Add Include Keyword </button>
    </div>

    <!-- Exclude Keywords List -->
    <div class="form-subsection">
      <h4>Exclude Keywords</h4>
      {#if internalFilter.exclude && internalFilter.exclude.length > 0}
        <ul class="list-items keyword-list">
          {#each internalFilter.exclude as keyword, index (index)}
            <ListItem item={keyword} {index} draggable={false} edit={() => openEditKeywordModal("exclude", index)} del={() => deleteKeyword("exclude", index)}>
              {keyword}
            </ListItem>
          {/each}
        </ul>
      {:else}
        <p class="empty-list-message">No exclude keywords.</p>
      {/if}
      <button type="button" class="button secondary-button add-item-button" onclick={() => openAddKeywordModal("exclude")}> Add Exclude Keyword </button>
    </div>

    <!-- Remove Section Button -->
    <div class="section-actions">
      <button type="button" class="button danger-button" onclick={removeFilterSection}> Remove Filter Section </button>
    </div>
  {/if}
</div>
