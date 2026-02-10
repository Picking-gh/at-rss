<script lang="ts">
  import { onMount } from "svelte";
  import Modal from "./components/Modal.svelte";
  import TaskDetail from "./components/TaskDetail.svelte";

  // --- State ---
  let tasks: { [key: string]: any } = $state({}); // Store fetched tasks {name: config}
  let selectedTaskName: string | null = $state(null);
  let isLoading = $state(true);
  let showNameInputModal = $state(false);
  let newTaskName = $state("");
  let nameInputError = $state("");
  let token = $state(localStorage.getItem("apiToken") || "");
  let showTokenModal = $state(false);
  let tokenInput = $state("");
  let tokenError = $state("");
  
  // Derived state: check if selected task is a new task
  let isAddingTask = $derived(selectedTaskName ? tasks[selectedTaskName]?.isNew || false : false);

  // --- API Helper ---
  async function apiFetch(url: string, options: RequestInit = {}) {
    try {
      // Add Authorization header if token exists
      const headers = new Headers(options.headers);
      if (token) {
        headers.set("Authorization", `Bearer ${token}`);
      }

      const response = await fetch(url, { ...options, headers });

      // Handle 401 Unauthorized
      if (response.status === 401) {
        showTokenModal = true;
        throw new Error("Unauthorized - Please provide a valid token");
      }

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`HTTP error! status: ${response.status}, message: ${errorText}`);
      }

      const contentType = response.headers.get("content-type");
      return contentType?.includes("application/json") ? await response.json() : await response.text();
    } catch (err: any) {
      console.error("API Fetch Error:", err);
      throw err; // Re-throw to allow specific handling if needed
    } finally {
      isLoading = false;
    }
  }

  function saveToken() {
    if (!tokenInput.trim()) {
      tokenError = "Token cannot be empty";
      return;
    }
    token = tokenInput;
    localStorage.setItem("apiToken", token);
    showTokenModal = false;
    tokenInput = "";
    tokenError = "";
    // Reload tasks after setting token
    loadTasks();
  }

  // --- Task Loading ---
  async function loadTasks() {
    isLoading = true;
    let rawTasks: any = null; // Temporary variable for the raw response
    try {
      rawTasks = await apiFetch("/api/tasks");

      // Explicitly check if the response is a valid, non-null object
      if (rawTasks && typeof rawTasks === "object" && !Array.isArray(rawTasks)) {
        // Sort tasks alphabetically by name for consistent display
        const sortedTaskNames = Object.keys(rawTasks).sort();
        const sortedTasks: { [key: string]: any } = {};
        for (const name of sortedTaskNames) {
          sortedTasks[name] = rawTasks[name];
        }
        tasks = sortedTasks;
      } else {
        // If response is not a valid object (e.g., null, [], undefined, etc.), treat as no tasks
        console.warn("Received non-object or null/empty response for tasks, setting tasks to empty object. Response:", rawTasks);
        tasks = {};
      }
    } catch (err) {
      // Error is handled in apiFetch, but ensure tasks is reset here too
      tasks = {};
    }
  }

  // --- Task Selection / Adding ---
  function selectTask(taskName: string) {
    selectedTaskName = taskName;
    // isAddingTask is now derived from the selected task's isNew property
  }

  // --- Adding Task ---
  function showNewTaskForm() {
    // Don't cancel existing new task, just show the name input modal
    // The existing new task will remain in the task list
    selectedTaskName = null; // Deselect any current task
    newTaskName = "";
    nameInputError = "";
    showNameInputModal = true;
  }

  function checkTaskNameExists(name: string): boolean {
    return Object.keys(tasks).some((taskName) => taskName.toLowerCase() === name.toLowerCase());
  }

  function handleTaskNameSubmit() {
    if (!newTaskName.trim()) {
      nameInputError = "Task name cannot be empty.";
      return;
    }

    if (checkTaskNameExists(newTaskName)) {
      nameInputError = `Task "${newTaskName}" already exists.`;
      return;
    }

    // Create new task config with isNew flag
    tasks[newTaskName] = {
      interval: null,
      downloaders: [],
      feeds: [],
      filter: null,
      extracter: null,
      isNew: true,
      isModified: false,
    };

    showNameInputModal = false;
    selectedTaskName = newTaskName;
    nameInputError = "";
  }

  // --- Event Handlers from TaskDetail ---
  function onTaskSaved(taskName: string) {
    if (tasks[taskName]) {
      tasks[taskName].isModified = false;
    }
  }

  function onTaskDeleted(taskName: string) {
    if (tasks[taskName]) {
      selectedTaskName = null;
      delete tasks[taskName];
      tasks = { ...tasks };
    }
  }

  function onTaskCreated(taskName: string) {
    if (tasks[taskName]) {
      tasks[taskName].isNew = false;
      tasks[taskName].isModified = false;
    }
  }

  function onNewTaskCanceled(taskName: string) {
    if (tasks[taskName]) {
      // If we're canceling the currently selected task, deselect it
      if (selectedTaskName === taskName) {
        selectedTaskName = null;
      }
      delete tasks[taskName];
      tasks = { ...tasks };
    }
    // Reset new task name for next use
    newTaskName = "";
    nameInputError = "";
  }

  function onTaskModified({ taskName, taskConfig, isModified }: { taskName: string; taskConfig: any; isModified: boolean }) {
    if (tasks[taskName]) {
      tasks[taskName] = taskConfig;
      tasks[taskName].isModified = isModified;
    }
  }

  // --- Lifecycle ---
  onMount(loadTasks);
</script>

<main class="app-container main-layout">
  <Modal showModal={showNameInputModal} title="New Task" close={() => {
    showNameInputModal = false;
    newTaskName = "";
    nameInputError = "";
  }}>
    {#snippet body()}
      <div class="form-group">
        <label for="task-name-input">Task Name</label>
        <input id="task-name-input" type="text" bind:value={newTaskName} class:error={nameInputError} class="modal-input" placeholder="Input a task name" />
        {#if nameInputError}
          <p class="error-message">{nameInputError}</p>
        {/if}
      </div>
    {/snippet}
    {#snippet footer()}
      <button type="button" class="button primary-button" onclick={handleTaskNameSubmit}>OK</button>
      <button type="button" class="button secondary-button" onclick={() => {
        showNameInputModal = false;
        newTaskName = "";
        nameInputError = "";
      }}>Cancel</button>
    {/snippet}
  </Modal>

  <Modal showModal={showTokenModal} title="API Authentication" close={() => {
    showTokenModal = false;
    tokenInput = "";
    tokenError = "";
  }}>
    {#snippet body()}
      <div class="form-group">
        <label for="token-input">API Token</label>
        <input id="token-input" type="password" bind:value={tokenInput} class:error={tokenError} class="modal-input" placeholder="Enter your API token" />
        {#if tokenError}
          <p class="error-message">{tokenError}</p>
        {/if}
      </div>
    {/snippet}
    {#snippet footer()}
      <button type="button" class="button primary-button" onclick={saveToken}>Save</button>
      <button type="button" class="button secondary-button" onclick={() => {
        showTokenModal = false;
        tokenInput = "";
        tokenError = "";
      }}>Cancel</button>
    {/snippet}
  </Modal>

   <aside class="sidebar task-list-panel">
     <h2>Tasks</h2>
     {#if isLoading}
       <div class="empty-state">
         <div class="loading-spinner large"></div>
         <p class="empty-state-title">Loading tasks...</p>
       </div>
     {:else if Object.keys(tasks).length === 0}
       <div class="empty-state">
         <div class="empty-state-icon">📋</div>
         <p class="empty-state-title">No tasks yet</p>
         <p class="empty-state-description">Create your first task to start monitoring RSS feeds</p>
         <button id="add-task-btn" class="button primary-button" onclick={() => showNewTaskForm()} style="margin-top: var(--spacing-md);">Create First Task</button>
       </div>
     {:else}
       <ul id="task-list">
         {#each Object.keys(tasks) as taskName (taskName)}
           <li>
             <button
               class="task-button"
               class:active={taskName === selectedTaskName}
               class:new-task={tasks[taskName]?.isNew}
               class:modified-task={tasks[taskName]?.isModified && !tasks[taskName]?.isNew}
               onclick={() => selectTask(taskName)}
             >
               {taskName}
             </button>
           </li>
         {/each}
         <li class="add-task-item">
           <button id="add-task-btn" class="button task-button add-button" onclick={() => showNewTaskForm()}> + Add New Task</button>
         </li>
       </ul>
     {/if}
   </aside>

    <section class="main-content task-detail-panel">
      <h2>
        {#if isAddingTask}
          {newTaskName || "New Task"} Details
        {:else if selectedTaskName}
          {selectedTaskName} Details
        {:else}
          Details
        {/if}
      </h2>
      {#if isAddingTask}
        <TaskDetail
          isNew={true}
          taskName={newTaskName}
          taskConfig={tasks[newTaskName]}
          {apiFetch}
          onTaskSaved={onTaskCreated}
          {onNewTaskCanceled}
          {onTaskModified}
        />
      {:else if selectedTaskName && tasks[selectedTaskName]}
        <TaskDetail isNew={false} taskName={selectedTaskName} taskConfig={tasks[selectedTaskName]} {apiFetch} {onTaskSaved} {onTaskDeleted} {onTaskModified} />
      {:else if selectedTaskName}
        <div class="empty-state">
          <div class="loading-spinner"></div>
          <p class="empty-state-title">Loading task details...</p>
        </div>
      {:else}
        <div class="empty-state">
          <div class="empty-state-icon">👈</div>
          <p class="empty-state-title">Select a task</p>
          <p class="empty-state-description">Choose a task from the sidebar to view or edit its configuration</p>
        </div>
      {/if}
    </section>
</main>
