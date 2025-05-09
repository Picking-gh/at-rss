<script lang="ts">
  import { onMount } from "svelte";
  import Modal from "./components/Modal.svelte";
  import TaskDetail from "./components/TaskDetail.svelte";

  // --- State ---
  let tasks: { [key: string]: any } = $state({}); // Store fetched tasks {name: config}
  let selectedTaskName: string | null = $state(null);
  let isAddingTask = $state(false); // Flag for adding a new task
  let isLoading = $state(true);
  let showNameInputModal = $state(false);
  let newTaskName = $state("");
  let nameInputError = $state("");
  let token = $state(localStorage.getItem("apiToken") || "");
  let showTokenModal = $state(false);
  let tokenInput = $state("");
  let tokenError = $state("");

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
        sortedTaskNames.forEach((name) => (sortedTasks[name] = rawTasks[name]));
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
    isAddingTask = tasks[taskName].isNew || false;
  }

  // --- Adding Task ---
  function showNewTaskForm() {
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
    isAddingTask = true;
    selectedTaskName = newTaskName;
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
      isAddingTask = false;
      tasks[taskName].isNew = false;
      tasks[taskName].isModified = false;
    }
  }

  function onNewTaskCanceled(taskName: string) {
    if (tasks[taskName]) {
      selectedTaskName = null;
      isAddingTask = false;
      delete tasks[taskName];
      tasks = { ...tasks };
    }
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
  <!-- Task Name Input Modal -->
  <Modal showModal={showNameInputModal} title="New Task" close={() => (showNameInputModal = false)}>
    {#snippet body()}
      <div class="form-group">
        <label for="task-name-input">Task Name</label>
        <input id="task-name-input" type="text" bind:value={newTaskName} class:error={nameInputError} placeholder="Input a task name" />
        {#if nameInputError}
          <p class="error-message">{nameInputError}</p>
        {/if}
      </div>
    {/snippet}
    {#snippet footer()}
      <button type="button" class="button primary-button" onclick={handleTaskNameSubmit}>OK</button>
      <button type="button" class="button secondary-button" onclick={() => (showNameInputModal = false)}>Cancel</button>
    {/snippet}
  </Modal>

  <!-- Token Input Modal -->
  <Modal showModal={showTokenModal} title="API Authentication" close={() => (showTokenModal = false)}>
    {#snippet body()}
      <div class="form-group">
        <label for="token-input">API Token</label>
        <input id="token-input" type="password" bind:value={tokenInput} class:error={tokenError} placeholder="Enter your API token" />
        {#if tokenError}
          <p class="error-message">{tokenError}</p>
        {/if}
      </div>
    {/snippet}
    {#snippet footer()}
      <button type="button" class="button primary-button" onclick={saveToken}>Save</button>
      <button type="button" class="button secondary-button" onclick={() => (showTokenModal = false)}>Cancel</button>
    {/snippet}
  </Modal>

  <aside class="sidebar task-list-panel">
    <h2>Tasks</h2>
    {#if isLoading}
      <p>Loading tasks...</p>
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
          <button id="add-task-btn" class="button task-button add-button" onclick={() => showNewTaskForm()}> + </button>
        </li>
      </ul>
    {/if}
  </aside>

  <section class="main-content task-detail-panel">
    <h2>Details</h2>
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
      <p>Loading task details for {selectedTaskName}... or task data missing.</p>
    {:else}
      <p>Please select a task from the list or add a new one.</p>
    {/if}
  </section>
</main>
