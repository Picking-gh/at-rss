<script lang="ts">
  import DownloaderSection from "./DownloaderSection.svelte";
  import FeedSection from "./FeedSection.svelte";
  import FilterSection from "./FilterSection.svelte";
  import ExtracterSection from "./ExtracterSection.svelte";
  import type { DownloaderConfig } from "../types";
  import type { FilterConfig } from "../types";
  import type { ExtracterConfig } from "../types";

  interface Props {
    taskConfig: any; // The configuration object for the selected task
    taskName: string; // The name of the selected task
    isNew?: boolean; // Flag indicating if it's a new task form
    apiFetch: (url: string, options?: RequestInit) => Promise<any>;
    onTaskModified?: (modifiedTask: { taskName: string; taskConfig: any; isModified: boolean }) => void;
    onTaskSaved?: (taskName: string) => void;
    onTaskDeleted?: (taskName: string) => void;
    onNewTaskCanceled?: (taskName: string) => void;
  }

  let { taskConfig, taskName, isNew = false, apiFetch, onTaskModified, onTaskSaved, onTaskDeleted, onNewTaskCanceled }: Props = $props();

  let internalTaskConfig = $state(taskConfig); // Deep copy to avoid modifying original object directly

  // --- Event Handlers ---

  function handleInputChange() {
    onTaskModified?.({ taskName: taskName, taskConfig: internalTaskConfig, isModified: true });
  }

  function handleDownloaderUpdate(data: DownloaderConfig) {
    internalTaskConfig.downloaders = data;
    handleInputChange();
  }

  function handleFeedUpdate(data: string[]) {
    internalTaskConfig.feeds = data;
    handleInputChange();
  }

  function handleFilterUpdate(data: FilterConfig) {
    internalTaskConfig.filter = data;
    handleInputChange();
  }

  function handleExtracterUpdate(data: ExtracterConfig) {
    internalTaskConfig.extracter = data;
    handleInputChange();
  }

  async function handleSave(event: Event) {
    event.preventDefault();
    try {
      const method = isNew ? "POST" : "PUT";
      const url = isNew ? "/api/tasks" : `/api/tasks/${taskName}`;
      const bodyPayload = isNew ? { config: internalTaskConfig, name: taskName } : internalTaskConfig;
      const body = JSON.stringify(bodyPayload);
      await apiFetch(url, { method, headers: { "Content-Type": "application/json" }, body });
      onTaskSaved?.(taskName);
    } catch (error: any) {
      alert(`Failed to save task: ${error.message}`);
      console.error("Save Task Error:", error);
    }
  }

  async function handleDelete(event: Event) {
    event.preventDefault();
    if (confirm(`Are you sure you want to delete task "${taskName}"?`)) {
      try {
        await apiFetch(`/api/tasks/${taskName}`, { method: "DELETE" });
        onTaskDeleted?.(taskName);
      } catch (error: any) {
        alert(`Failed to delete task: ${error.message}`);
        console.error("Delete Task Error:", error);
      }
    }
  }

  // Reset internal state
  $effect(() => {
    internalTaskConfig = taskConfig;
  });
</script>

<div id="task-form-container" class="task-detail-container">
  <form onsubmit={handleSave}>
    <!-- Basic Info Section -->
    <div class="form-section">
      <h3>Basic Info</h3>
      <div class="form-group">
        <label for="taskNameInput">Task Name</label>
        <input type="text" id="taskNameInput" value={taskName} readonly style="background-color: #eee;" />
      </div>
      <div class="form-group">
        <label for="interval">Fetch Interval (minutes)</label>
        <input type="number" id="interval" bind:value={internalTaskConfig.interval} min="1" placeholder="e.g., 10" oninput={handleInputChange} />
      </div>
    </div>

    <!-- Downloader List Section -->
    <DownloaderSection downloaders={internalTaskConfig.downloaders} update={handleDownloaderUpdate} />

    <!-- Feed List Section -->
    <FeedSection feeds={internalTaskConfig.feeds} update={handleFeedUpdate} />

    <!-- Filter Section -->
    <FilterSection filter={internalTaskConfig.filter} update={handleFilterUpdate} />

    <!-- Extracter Section -->
    <ExtracterSection extracter={internalTaskConfig.extracter} update={handleExtracterUpdate} />

    <!-- Action Buttons -->
    <div class="action-buttons">
      {#if taskConfig?.isModified || isNew}
        <button type="submit" class="button primary-button">
          {isNew ? "Create Task" : "Save Changes"}
        </button>
      {/if}
      {#if !isNew}
        <button type="button" class="button danger-button" onclick={handleDelete}> Delete Task </button>
      {/if}
      {#if isNew}
        <button type="button" class="button secondary-button" onclick={() => onNewTaskCanceled?.(taskName)}> Cancel </button>
      {/if}
    </div>
  </form>
</div>
