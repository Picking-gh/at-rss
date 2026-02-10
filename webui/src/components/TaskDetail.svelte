<script lang="ts">
  import DownloaderSection from "./DownloaderSection.svelte";
  import FeedSection from "./FeedSection.svelte";
  import FilterSection from "./FilterSection.svelte";
  import ExtracterSection from "./ExtracterSection.svelte";
  import type { DownloaderConfig } from "../types";
  import type { FilterConfig } from "../types";
  import type { ExtracterConfig } from "../types";

  interface TaskConfig {
    interval?: number | null;
    downloaders?: DownloaderConfig[];
    feeds?: string[];
    filter?: FilterConfig | null;
    extracter?: ExtracterConfig | null;
    isNew?: boolean;
    isModified?: boolean;
  }

  interface Props {
    taskConfig: TaskConfig; // The configuration object for the selected task
    taskName: string; // The name of the selected task
    isNew?: boolean; // Flag indicating if it's a new task form
    apiFetch: (url: string, options?: RequestInit) => Promise<any>;
    onTaskModified?: (modifiedTask: { taskName: string; taskConfig: TaskConfig; isModified: boolean }) => void;
    onTaskSaved?: (taskName: string) => void;
    onTaskDeleted?: (taskName: string) => void;
    onNewTaskCanceled?: (taskName: string) => void;
  }

  let { taskConfig, taskName, isNew = false, apiFetch, onTaskModified, onTaskSaved, onTaskDeleted, onNewTaskCanceled }: Props = $props();

  let internalTaskConfig: TaskConfig = $state({
    interval: null,
    downloaders: [],
    feeds: [],
    filter: null,
    extracter: null,
    isNew: false,
    isModified: false
  });
  
  // Update internal state when prop changes
  $effect(() => {
    // Ensure taskConfig has all required properties
    internalTaskConfig = {
      interval: taskConfig?.interval ?? null,
      downloaders: taskConfig?.downloaders ?? [],
      feeds: taskConfig?.feeds ?? [],
      filter: taskConfig?.filter ?? null,
      extracter: taskConfig?.extracter ?? null,
      isNew: taskConfig?.isNew ?? false,
      isModified: taskConfig?.isModified ?? false
    };
  });

  // --- Event Handlers ---

  function handleInputChange() {
    onTaskModified?.({ taskName: taskName, taskConfig: internalTaskConfig, isModified: true });
  }

  function handleDownloaderUpdate(data: DownloaderConfig[]) {
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



   // Tab state
   let activeTab = $state('downloaders');
   
   function setActiveTab(tab: string) {
     activeTab = tab;
   }
 </script>

<div id="task-form-container" class="task-detail-container">
  <form onsubmit={handleSave} class="tab-container">
    <div class="tab-header">
      <button type="button" class:active={activeTab === 'downloaders'} class="tab-button" onclick={() => setActiveTab('downloaders')}>
        Downloaders
      </button>
      <button type="button" class:active={activeTab === 'feeds'} class="tab-button" onclick={() => setActiveTab('feeds')}>
        Feeds
      </button>
      <button type="button" class:active={activeTab === 'include'} class="tab-button" onclick={() => setActiveTab('include')}>
        Include
      </button>
      <button type="button" class:active={activeTab === 'exclude'} class="tab-button" onclick={() => setActiveTab('exclude')}>
        Exclude
      </button>
      <button type="button" class:active={activeTab === 'extracter'} class="tab-button" onclick={() => setActiveTab('extracter')}>
        Extracter
      </button>
    </div>

    <div class="tab-content">
      <div class:active={activeTab === 'downloaders'} class="tab-pane" data-tab-title="Downloaders">
        <div class="tab-form-section">
          <DownloaderSection downloaders={internalTaskConfig.downloaders} update={handleDownloaderUpdate} />
        </div>
      </div>

      <div class:active={activeTab === 'feeds'} class="tab-pane" data-tab-title="Feeds">
        <div class="tab-form-section">
          <FeedSection feeds={internalTaskConfig.feeds} update={handleFeedUpdate} />
        </div>
      </div>

      <div class:active={activeTab === 'include'} class="tab-pane" data-tab-title="Include Filters">
        <div class="tab-form-section">
          <FilterSection filter={internalTaskConfig.filter} type="include" update={handleFilterUpdate} />
        </div>
      </div>

      <div class:active={activeTab === 'exclude'} class="tab-pane" data-tab-title="Exclude Filters">
        <div class="tab-form-section">
          <FilterSection filter={internalTaskConfig.filter} type="exclude" update={handleFilterUpdate} />
        </div>
      </div>

      <div class:active={activeTab === 'extracter'} class="tab-pane" data-tab-title="Extracter">
        <div class="tab-form-section">
          <ExtracterSection extracter={internalTaskConfig.extracter} update={handleExtracterUpdate} />
        </div>
      </div>
    </div>

    <div class="bottom-action-bar">
      <div class="bottom-action-left">
        <div class="interval-control">
          <label for="interval">Interval (minutes):</label>
          <input type="number" id="interval" bind:value={internalTaskConfig.interval} min="1" placeholder="10" oninput={handleInputChange} />
        </div>
      </div>
      <div class="bottom-action-spacer"></div>
      <div class="bottom-action-right">
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
    </div>
  </form>
</div>
