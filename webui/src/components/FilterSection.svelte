<script lang="ts">
  import Modal from "./Modal.svelte";
  import ListItem from "./ListItem.svelte";
  import type { FilterConfig } from "../types";

  interface Props {
    filter?: FilterConfig | null | undefined; // The filter configuration object
    type?: "include" | "exclude"; // Which type of filter to show
    update?: any;
  }

  let { filter = null, type = "include", update }: Props = $props();

  // --- Local State ---
  // Initialize with empty filter, update when prop changes
  let internalFilter: FilterConfig = $state({ include: [], exclude: [] });

  // Update internal state when prop changes
  $effect(() => {
    if (filter) {
      internalFilter = filter;
    } else {
      internalFilter = { include: [], exclude: [] };
    }
  });

  // --- Modal State ---
  let showKeywordModal = $state(false);
  let modalTitle = $state("");
  let currentKeywordValue = $state("");
  let editingKeywordType: "include" | "exclude" | null = null;
  let editingKeywordIndex: number | null = null;

  // --- Event Handlers ---
  function notifyUpdate() {
    // Check if both include and exclude arrays are empty
    const includeEmpty = !internalFilter.include || internalFilter.include.length === 0;
    const excludeEmpty = !internalFilter.exclude || internalFilter.exclude.length === 0;
    
    if (includeEmpty && excludeEmpty) {
      // Both arrays are empty, set filter to null
      update(null);
    } else {
      // At least one array has content, create filter object
      const updatedFilter: FilterConfig = {
        include: internalFilter.include || [],
        exclude: internalFilter.exclude || [],
      };
      update(updatedFilter);
    }
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


</script>

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
  {#if internalFilter[type] && internalFilter[type].length > 0}
    <ul class="list-items keyword-list">
      {#each internalFilter[type] as keyword, index (index)}
        <ListItem item={keyword} {index} draggable={false} edit={() => openEditKeywordModal(type, index)} del={() => deleteKeyword(type, index)}>
          {keyword}
        </ListItem>
      {/each}
    </ul>
  {:else}
    <p class="empty-list-message">No {type} keywords.</p>
  {/if}
  <button type="button" class="button secondary-button add-item-button" onclick={() => openAddKeywordModal(type)}>
    Add {type === "include" ? "Include" : "Exclude"} Keyword
  </button>
</div>
