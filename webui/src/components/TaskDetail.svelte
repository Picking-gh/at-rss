<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import DownloaderSection from "./DownloaderSection.svelte";
  import FeedSection from "./FeedSection.svelte";
  import FilterSection from "./FilterSection.svelte";
  import ExtracterSection from "./ExtracterSection.svelte"; // Import ExtracterSection

  export let taskConfig: any; // The configuration object for the selected task
  export let taskName: string; // The name of the selected task
  export let isNew: boolean = false; // Flag indicating if it's a new task form
  export let apiFetch: (url: string, options?: RequestInit) => Promise<any>; // Function passed from parent

  const dispatch = createEventDispatcher();

  let internalTaskConfig = structuredClone(taskConfig); // Deep copy to avoid modifying original object directly
  let newTaskName: string = ""; // Store the name for a new task
  let isModified = taskConfig?.isModified || isNew || false; // Track local modifications

  // --- Form Data Handling ---
  // We'll bind form elements directly to internalTaskConfig properties

  // --- Event Handlers ---
  function handleInputChange() {
    if (!isModified) {
      isModified = true; // Mark as modified on first change
      // Optionally, update the parent's task state immediately or wait for save
      // dispatch('update:task', { taskName, config: { ...internalTaskConfig, isModified: true } });
    }
    // TODO: Add more sophisticated change detection if needed
  }

  function handleDownloaderUpdate(event: CustomEvent) {
    internalTaskConfig.downloaders = event.detail;
    handleInputChange(); // Mark task as modified
  }

  function handleFeedUpdate(event: CustomEvent) {
    internalTaskConfig.feeds = event.detail;
    handleInputChange(); // Mark task as modified
  }

  function handleFilterUpdate(event: CustomEvent) {
    internalTaskConfig.filter = event.detail;
    handleInputChange(); // Mark task as modified
  }

  function handleExtracterUpdate(event: CustomEvent) {
    internalTaskConfig.extracter = event.detail;
    handleInputChange(); // Mark task as modified
  }

  async function handleSave() {
    const nameToSave = isNew ? newTaskName.trim() : taskName;
    if (isNew && !nameToSave) {
      alert("Task name cannot be empty.");
      return;
    }

    console.log("Saving task:", nameToSave, internalTaskConfig);
    try {
      const method = isNew ? "POST" : "PUT";
      const url = isNew ? "/api/tasks" : `/api/tasks/${nameToSave}`; // Use nameToSave
      // For new tasks, the name is part of the body or handled by the API endpoint structure
      const bodyPayload = isNew ? { config: internalTaskConfig, name: nameToSave } : internalTaskConfig;
      const body = JSON.stringify(bodyPayload);

      await apiFetch(url, { method, headers: { "Content-Type": "application/json" }, body });

      dispatch("taskSaved", { taskName: nameToSave, isNew }); // Notify parent with the correct name
      isModified = false; // Reset modified state after successful save
      if (isNew) {
        newTaskName = ""; // Clear the input for the next potential new task
      }
    } catch (error: any) {
      alert(`Failed to save task: ${error.message}`);
      console.error("Save Task Error:", error);
    }
  }

  async function handleDelete() {
    if (confirm(`Are you sure you want to delete task "${taskName}"?`)) {
      console.log("Deleting task:", taskName);
      try {
        await apiFetch(`/api/tasks/${taskName}`, { method: "DELETE" });
        dispatch("taskDeleted", { taskName }); // Notify parent
      } catch (error: any) {
        alert(`Failed to delete task: ${error.message}`);
        console.error("Delete Task Error:", error);
      }
    }
  }

  // Reset internal state when the input taskConfig changes or when switching to/from 'new' mode
  $: {
    internalTaskConfig = structuredClone(taskConfig);
    isModified = taskConfig?.isModified || isNew || false;
    if (isNew) {
      newTaskName = ""; // Reset new task name when isNew becomes true
    }
  }
</script>

<div id="task-form-container" class="task-detail-container">
  <form on:submit|preventDefault={handleSave}>
    <!-- Basic Info Section -->
    <div class="form-section">
      <h3>Basic Info</h3>
      <div class="form-group">
        <label for="taskNameInput">Task Name</label>
        {#if isNew}
          <input type="text" id="taskNameInput" bind:value={newTaskName} placeholder="Enter a unique task name" required on:input={handleInputChange} />
        {:else}
          <input type="text" id="taskNameInput" value={taskName} readonly style="background-color: #eee;" />
        {/if}
      </div>
      <div class="form-group">
        <label for="interval">Fetch Interval (minutes)</label>
        <input type="number" id="interval" bind:value={internalTaskConfig.interval} min="1" placeholder="e.g., 10" on:input={handleInputChange} />
      </div>
    </div>

    <!-- Downloader List Section -->
    <DownloaderSection bind:downloaders={internalTaskConfig.downloaders} {apiFetch} on:update:downloaders={handleDownloaderUpdate} />

    <!-- Feed List Section -->
    <FeedSection bind:feeds={internalTaskConfig.feeds} on:update:feeds={handleFeedUpdate} />

    <!-- Filter Section -->
    <FilterSection bind:filter={internalTaskConfig.filter} on:update:filter={handleFilterUpdate} />

    <!-- Extracter Section -->
    <ExtracterSection bind:extracter={internalTaskConfig.extracter} on:update:extracter={handleExtracterUpdate} />

    <!-- Action Buttons -->
    <div class="action-buttons">
      {#if isModified || isNew}
        <button type="submit" class="button primary-button">
          {isNew ? "Create Task" : "Save Changes"}
        </button>
      {/if}
      {#if !isNew}
        <button type="button" class="button danger-button" on:click={handleDelete}> Delete Task </button>
      {/if}
      {#if isNew}
        <button type="button" class="button secondary-button" on:click={() => dispatch("cancelAdd")}> Cancel </button>
      {/if}
    </div>
  </form>
</div>
