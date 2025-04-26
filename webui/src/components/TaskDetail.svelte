<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import DownloaderSection from "./DownloaderSection.svelte";
  import FeedSection from "./FeedSection.svelte";
  import FilterSection from "./FilterSection.svelte";
  import ExtracterSection from "./ExtracterSection.svelte";

  export let taskConfig: any; // The configuration object for the selected task
  export let taskName: string; // The name of the selected task
  export let isNew: boolean = false; // Flag indicating if it's a new task form
  export let apiFetch: (url: string, options?: RequestInit) => Promise<any>; // Function passed from parent

  const dispatch = createEventDispatcher();

  let internalTaskConfig = structuredClone(taskConfig); // Deep copy to avoid modifying original object directly

  // --- Event Handlers ---

  function handleInputChange() {
    dispatch("update:modified", {
      taskName,
      taskConfig: internalTaskConfig,
      isModified: true,
    });
  }

  function handleDownloaderUpdate(event: CustomEvent) {
    internalTaskConfig.downloaders = event.detail;
    handleInputChange();
  }

  function handleFeedUpdate(event: CustomEvent) {
    internalTaskConfig.feeds = event.detail;
    handleInputChange();
  }

  function handleFilterUpdate(event: CustomEvent) {
    internalTaskConfig.filter = event.detail;
    handleInputChange();
  }

  function handleExtracterUpdate(event: CustomEvent) {
    internalTaskConfig.extracter = event.detail;
    handleInputChange();
  }

  async function handleSave() {
    try {
      const method = isNew ? "POST" : "PUT";
      const url = isNew ? "/api/tasks" : `/api/tasks/${taskName}`;
      // For new tasks, the name is part of the body or handled by the API endpoint structure
      const bodyPayload = isNew ? { config: internalTaskConfig, name: taskName } : internalTaskConfig;
      const body = JSON.stringify(bodyPayload);

      await apiFetch(url, { method, headers: { "Content-Type": "application/json" }, body });

      dispatch("taskSaved", { taskName: taskName });
    } catch (error: any) {
      alert(`Failed to save task: ${error.message}`);
      console.error("Save Task Error:", error);
    }
  }

  async function handleDelete() {
    if (confirm(`Are you sure you want to delete task "${taskName}"?`)) {
      try {
        await apiFetch(`/api/tasks/${taskName}`, { method: "DELETE" });
        dispatch("taskDeleted", { taskName });
      } catch (error: any) {
        alert(`Failed to delete task: ${error.message}`);
        console.error("Delete Task Error:", error);
      }
    }
  }

  // Reset internal state
  $: internalTaskConfig = structuredClone(taskConfig);
</script>

<div id="task-form-container" class="task-detail-container">
  <form on:submit|preventDefault={handleSave}>
    <!-- Basic Info Section -->
    <div class="form-section">
      <h3>Basic Info</h3>
      <div class="form-group">
        <label for="taskNameInput">Task Name</label>
        <input type="text" id="taskNameInput" value={taskName} readonly style="background-color: #eee;" />
      </div>
      <div class="form-group">
        <label for="interval">Fetch Interval (minutes)</label>
        <input type="number" id="interval" bind:value={internalTaskConfig.interval} min="1" placeholder="e.g., 10" on:input={handleInputChange} />
      </div>
    </div>

    <!-- Downloader List Section -->
    <DownloaderSection bind:downloaders={internalTaskConfig.downloaders} on:update:downloaders={handleDownloaderUpdate} />

    <!-- Feed List Section -->
    <FeedSection bind:feeds={internalTaskConfig.feeds} on:update:feeds={handleFeedUpdate} />

    <!-- Filter Section -->
    <FilterSection bind:filter={internalTaskConfig.filter} on:update:filter={handleFilterUpdate} />

    <!-- Extracter Section -->
    <ExtracterSection bind:extracter={internalTaskConfig.extracter} on:update:extracter={handleExtracterUpdate} />

    <!-- Action Buttons -->
    <div class="action-buttons">
      {#if taskConfig?.isModified || isNew}
        <button type="submit" class="button primary-button">
          {isNew ? "Create Task" : "Save Changes"}
        </button>
      {/if}
      {#if !isNew}
        <button type="button" class="button danger-button" on:click={handleDelete}> Delete Task </button>
      {/if}
      {#if isNew}
        <button type="button" class="button secondary-button" on:click={() => dispatch("cancelAdd", { taskName })}> Cancel </button>
      {/if}
    </div>
  </form>
</div>
