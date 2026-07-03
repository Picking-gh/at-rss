<script lang="ts">
  import Modal from "./Modal.svelte";
  import ListItem from "./ListItem.svelte";
  import type { FilterConfig } from "../types";

  interface Props {
    filter?: FilterConfig | null | undefined;
    type?: "include" | "exclude";
    update?: any;
  }

  let { filter = null, type = "include", update }: Props = $props();

  let internalFilter: FilterConfig = $state({ include: [], exclude: [] });

  $effect(() => {
    if (filter) {
      internalFilter = filter;
    } else {
      internalFilter = { include: [], exclude: [] };
    }
  });

  let showKeywordModal = $state(false);
  let modalTitle = $state("");
  let currentKeywordValue = $state("");
  let editingKeywordType: "include" | "exclude" | null = null;
  let editingKeywordIndex: number | null = null;

  // Empty string placeholder = intentional "download nothing".
  // An empty string cannot be typed in the Web UI (input validation rejects it)
  // and can never match a real title in the backend.
  const NONE_PLACEHOLDER = "";

  function notifyUpdate() {
    const updatedFilter: FilterConfig = {
      include: internalFilter.include || [],
      exclude: internalFilter.exclude || [],
    };
    update(updatedFilter);
  }

  function hasNonePlaceholder(): boolean {
    const list = internalFilter[type];
    return !!list && list.length === 1 && list[0] === NONE_PLACEHOLDER;
  }

  function visibleKeywords(): string[] {
    const list = internalFilter[type] || [];
    return list.filter(k => k !== NONE_PLACEHOLDER);
  }

  function clearNonePlaceholder() {
    internalFilter[type] = [];
    notifyUpdate();
  }

  function openAddKeywordModal(t: "include" | "exclude") {
    modalTitle = `Add ${t.charAt(0).toUpperCase() + t.slice(1)} Keyword`;
    currentKeywordValue = "";
    editingKeywordType = t;
    editingKeywordIndex = null;
    showKeywordModal = true;
  }

  function openEditKeywordModal(t: "include" | "exclude", index: number) {
    const kw = visibleKeywords()[index];
    if (kw === undefined) return;
    modalTitle = `Edit ${t.charAt(0).toUpperCase() + t.slice(1)} Keyword`;
    currentKeywordValue = kw;
    editingKeywordType = t;
    editingKeywordIndex = index;
    showKeywordModal = true;
  }

  function saveKeyword() {
    const keywordValue = currentKeywordValue.trim();
    if (!keywordValue || !editingKeywordType) {
      alert("Keyword cannot be empty.");
      return;
    }
    // Remove placeholder when adding real keyword
    if (hasNonePlaceholder()) {
      internalFilter[editingKeywordType] = [];
    }
    const list = internalFilter[editingKeywordType] || [];

    if (editingKeywordIndex !== null) {
      const originalKeyword = list[editingKeywordIndex];
      if (keywordValue === originalKeyword) {
        showKeywordModal = false;
        return;
      }
      if (list.some((kw, i) => i !== editingKeywordIndex && kw === keywordValue)) {
        alert(`Keyword "${keywordValue}" already exists.`);
        return;
      }
      const updatedKeywords = [...list];
      updatedKeywords[editingKeywordIndex] = keywordValue;
      internalFilter[editingKeywordType] = updatedKeywords;
    } else {
      if (list.includes(keywordValue)) {
        alert(`Keyword "${keywordValue}" already exists.`);
        return;
      }
      internalFilter[editingKeywordType] = [...list, keywordValue];
    }
    notifyUpdate();
    showKeywordModal = false;
  }

  function deleteKeyword(t: "include" | "exclude", index: number) {
    const kw = visibleKeywords()[index];
    if (kw === undefined) return;
    const list = internalFilter[t] || [];
    const realIndex = list.indexOf(kw);
    if (realIndex < 0) return;

    if (t === "include" && visibleKeywords().length === 1) {
      // Last include keyword being deleted — prompt for intent
      showClearConfirm = true;
    } else {
      if (confirm(`Delete this ${t} keyword?`)) {
        internalFilter[t] = list.filter((_, i) => i !== realIndex);
        notifyUpdate();
      }
    }
  }

  // State for the "last keyword deleted" confirmation
  let showClearConfirm = $state(false);

  function confirmDownloadAll() {
    internalFilter.include = [];
    showClearConfirm = false;
    notifyUpdate();
  }

  function confirmDownloadNothing() {
    internalFilter.include = [NONE_PLACEHOLDER];
    showClearConfirm = false;
    notifyUpdate();
  }

  function cancelClearConfirm() {
    showClearConfirm = false;
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
  {#if showClearConfirm}
    <div class="filter-warning">
      <p><strong>Include filter is now empty.</strong> What should happen?</p>
      <div class="confirm-buttons">
        <button type="button" class="button primary-button" onclick={confirmDownloadAll}>Download All Items</button>
        <button type="button" class="button danger-button" onclick={confirmDownloadNothing}>Download Nothing</button>
        <button type="button" class="button secondary-button" onclick={cancelClearConfirm}>Cancel</button>
      </div>
    </div>
  {:else if hasNonePlaceholder()}
    <div class="filter-warning">
      ⚠️ <strong>Nothing will be downloaded.</strong>
      <br/>Include filter is paused. To start downloading, remove this restriction.
      <div class="confirm-buttons" style="margin-top: 0.5rem;">
        <button type="button" class="button primary-button" onclick={clearNonePlaceholder}>Download All Items</button>
      </div>
    </div>
  {:else if !internalFilter[type] || internalFilter[type].length === 0}
    <p class="empty-list-message">
      No {type} keywords. {type === "include" ? "All items will be downloaded." : ""}
    </p>
  {:else}
    <ul class="list-items keyword-list">
      {#each visibleKeywords() as keyword, index (index)}
        <ListItem item={keyword} {index} draggable={false} edit={() => openEditKeywordModal(type, index)} del={() => deleteKeyword(type, index)}>
          {keyword}
        </ListItem>
      {/each}
    </ul>
  {/if}

  {#if !showClearConfirm && !hasNonePlaceholder()}
    <button type="button" class="button secondary-button add-item-button" onclick={() => openAddKeywordModal(type)}>
      Add {type === "include" ? "Include" : "Exclude"} Keyword
    </button>
  {/if}
</div>
