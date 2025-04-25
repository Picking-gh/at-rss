<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import Modal from "./Modal.svelte"; // Import Modal

  // Define the structure for the filter object
  interface FilterConfig {
    include?: string[];
    exclude?: string[];
  }

  export let filter: FilterConfig | null | undefined = null; // The filter configuration object

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

  // --- Local State ---
  // Use derived state or direct binding if possible, but for complex objects, local copy might be easier
  let internalFilter: FilterConfig = filter ? JSON.parse(JSON.stringify(filter)) : { include: [], exclude: [] };

  // Update internal state when the prop changes
  $: internalFilter = filter ? JSON.parse(JSON.stringify(filter)) : { include: [], exclude: [] };

  // --- Modal State ---
  let showKeywordModal = false;
  let modalTitle = "";
  let currentKeywordValue = "";
  let editingKeywordType: "include" | "exclude" | null = null;
  let editingKeywordIndex: number | null = null;

  // --- Event Handlers ---
  function notifyUpdate() {
    // Ensure include/exclude are always arrays, even if empty
    const updatedFilter: FilterConfig = {
      include: internalFilter.include || [],
      exclude: internalFilter.exclude || [],
    };
    dispatch("update:filter", updatedFilter);
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
    showKeywordModal = false; // Close modal
  }

  function deleteKeyword(type: "include" | "exclude", index: number) {
    if (confirm(`Are you sure you want to delete this ${type} keyword?`)) {
      internalFilter[type] = internalFilter[type]?.filter((_, i) => i !== index);
      notifyUpdate();
    }
  }

  function addFilterSection() {
    dispatch("update:filter", { include: [], exclude: [] }); // Dispatch a new empty filter object
  }

  function removeFilterSection() {
    if (confirm("Are you sure you want to remove the entire filter section?")) {
      dispatch("update:filter", null); // Dispatch null to remove the filter
    }
  }
</script>

<!-- Keyword Add/Edit Modal -->
<Modal bind:showModal={showKeywordModal} title={modalTitle} on:close={() => (showKeywordModal = false)}>
  <div slot="body">
    <div class="form-group">
      <label for="keyword-input">Keyword</label>
      <input type="text" id="keyword-input" bind:value={currentKeywordValue} placeholder="Enter keyword" required class="modal-input" />
    </div>
  </div>
  <div slot="footer">
    <button type="button" class="button secondary-button" on:click={() => (showKeywordModal = false)}>Cancel</button>
    <button type="button" class="button primary-button" on:click={saveKeyword}>Save Keyword</button>
  </div>
</Modal>

<div class="form-section">
  <h3>Filter</h3>
  {#if filter === null || filter === undefined}
    <button type="button" class="button secondary-button" on:click={addFilterSection}> Add Filter Section </button>
  {:else}
    <!-- Include Keywords List -->
    <div class="form-subsection">
      <h4>Include Keywords</h4>
      {#if internalFilter.include && internalFilter.include.length > 0}
        <ul class="list-items keyword-list">
          {#each internalFilter.include as keyword, index (index)}
            <li class="list-item">
              <span class="item-content">{keyword}</span>
              <div class="list-item-actions">
                <button type="button" class="button icon-button secondary-button" on:click={() => openEditKeywordModal("include", index)} title="Edit Keyword">
                  {@html EDIT_ICON_SVG}
                </button>
                <button type="button" class="button icon-button danger-button" on:click={() => deleteKeyword("include", index)} title="Delete Keyword">
                  {@html DELETE_ICON_SVG}
                </button>
              </div>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="empty-list-message">No include keywords.</p>
      {/if}
      <button type="button" class="button secondary-button add-item-button" on:click={() => openAddKeywordModal("include")}> Add Include Keyword </button>
    </div>

    <!-- Exclude Keywords List -->
    <div class="form-subsection">
      <h4>Exclude Keywords</h4>
      {#if internalFilter.exclude && internalFilter.exclude.length > 0}
        <ul class="list-items keyword-list">
          {#each internalFilter.exclude as keyword, index (index)}
            <li class="list-item">
              <span class="item-content">{keyword}</span>
              <div class="list-item-actions">
                <button type="button" class="button icon-button secondary-button" on:click={() => openEditKeywordModal("exclude", index)} title="Edit Keyword">
                  {@html EDIT_ICON_SVG}
                </button>
                <button type="button" class="button icon-button danger-button" on:click={() => deleteKeyword("exclude", index)} title="Delete Keyword">
                  {@html DELETE_ICON_SVG}
                </button>
              </div>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="empty-list-message">No exclude keywords.</p>
      {/if}
      <button type="button" class="button secondary-button add-item-button" on:click={() => openAddKeywordModal("exclude")}> Add Exclude Keyword </button>
    </div>

    <!-- Remove Section Button -->
    <div class="section-actions">
      <button type="button" class="button danger-button" on:click={removeFilterSection}> Remove Filter Section </button>
    </div>
  {/if}
</div>
