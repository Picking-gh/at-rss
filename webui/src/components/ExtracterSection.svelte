<script lang="ts">
  import type { ExtracterConfig } from "../types";

  interface Props {
    extracter?: ExtracterConfig | null | undefined; // The extracter configuration object
    update?: any;
  }

  let { extracter = null, update }: Props = $props();

  // --- Local State ---
  let internalExtracter: ExtracterConfig = $state(extracter ? extracter : {});

  // --- Event Handlers ---
  function handleInputChange() {
    update({ ...internalExtracter });
  }

  function addExtracterSection() {
    update({ tag: "link", pattern: "" });
  }

  function removeExtracterSection() {
    if (confirm("Are you sure you want to remove the entire extracter section?")) {
      update(null);
    }
  }
</script>

<div class="form-section">
  <h3>Extracter (CSS Selectors)</h3>
  {#if extracter === null || extracter === undefined}
    <button type="button" class="button secondary-button" onclick={addExtracterSection}> Add Extracter Section </button>
  {:else}
    <div class="form-group">
      <label for="extracter-tag">Tag</label>
      <select id="extracter-tag" bind:value={internalExtracter.tag} required oninput={handleInputChange}>
        <option value="title">title</option>
        <option value="link">link</option>
        <option value="description">description</option>
        <option value="enclosure">enclosure</option>
        <option value="guid">guid</option>
      </select>
    </div>

    <div class="form-group">
      <label for="extracter-pattern">Pattern (Regex)</label>
      <input
        type="text"
        id="extracter-pattern"
        bind:value={internalExtracter.pattern}
        placeholder="e.g., (?:[2-7A-Z]&#123;32&#125;|[0-9a-f]&#123;40&#125;)"
        oninput={handleInputChange}
      />
    </div>

    <!-- Remove Section Button -->
    <div class="section-actions">
      <button type="button" class="button danger-button" onclick={removeExtracterSection}> Remove Extracter Section </button>
    </div>
  {/if}
</div>
