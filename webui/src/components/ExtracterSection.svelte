<script lang="ts">
  import { createEventDispatcher } from "svelte";

  // Define the structure for the extracter object
  interface ExtracterConfig {
    tag?: "title" | "link" | "description" | "enclosure" | "guid";
    pattern?: string;
  }

  export let extracter: ExtracterConfig | null | undefined = null; // The extracter configuration object

  const dispatch = createEventDispatcher();

  // --- Local State ---
  let internalExtracter: ExtracterConfig = extracter ? JSON.parse(JSON.stringify(extracter)) : {};

  // Update internal state when the prop changes
  // $: internalExtracter = extracter ? JSON.parse(JSON.stringify(extracter)) : {};

  // --- Event Handlers ---
  function handleInputChange() {
    // Dispatch the entire internal object on any change
    dispatch("update:extracter", { ...internalExtracter });
  }

  function addExtracterSection() {
    dispatch("update:extracter", { tag: "link", pattern: "" }); // Dispatch a new empty object
  }

  function removeExtracterSection() {
    if (confirm("Are you sure you want to remove the entire extracter section?")) {
      dispatch("update:extracter", null); // Dispatch null to remove
    }
  }
</script>

<div class="form-section">
  <h3>Extracter (CSS Selectors)</h3>
  {#if extracter === null || extracter === undefined}
    <button type="button" class="button secondary-button" on:click={addExtracterSection}> Add Extracter Section </button>
  {:else}
    <div class="form-group">
      <label for="extracter-tag">Tag</label>
      <select id="extracter-tag" bind:value={internalExtracter.tag} required on:input={handleInputChange}>
        <option value="title">title</option>
        <option value="link">link</option>
        <option value="description">description</option>
        <option value="enclosure">enclosure</option>
        <option value="guid">guid</option>
        <!-- Add other types if supported -->
      </select>
      <small>Selector for the container of each item/entry.</small>
    </div>

    <div class="form-group">
      <label for="extracter-pattern">Pattern (Regex)</label>
      <input
        type="text"
        id="extracter-pattern"
        bind:value={internalExtracter.pattern}
        placeholder="e.g., magnet:\\?xt=urn:btih:([a-fA-F0-9]{40})"
        on:input={handleInputChange}
      />
      <small>Selector for the pattern within each item (relative to Item Selector).</small>
    </div>

    <!-- Remove Section Button -->
    <div class="section-actions">
      <button type="button" class="button danger-button" on:click={removeExtracterSection}> Remove Extracter Section </button>
    </div>
  {/if}
</div>

