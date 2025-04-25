<script lang="ts">
  import { onMount } from "svelte";
  import TaskDetail from "./components/TaskDetail.svelte"; // Import TaskDetail

  // --- State ---
  let tasks: { [key: string]: any } = {}; // Store fetched tasks {name: config}
  let selectedTaskName: string | null = null;
  let isAddingTask = false; // Flag for adding a new task
  let isLoading = true;
  let error: string | null = null;

  // --- API Helper ---
  async function apiFetch(url: string, options: RequestInit = {}) {
    try {
      const response = await fetch(url, options);
      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`HTTP error! status: ${response.status}, message: ${errorText}`);
      }
      const contentType = response.headers.get("content-type");
      return contentType?.includes("application/json") ? await response.json() : await response.text();
    } catch (err: any) {
      console.error("API Fetch Error:", err);
      error = err.message || "Failed to fetch data.";
      throw err; // Re-throw to allow specific handling if needed
    } finally {
      isLoading = false;
    }
  }

  // --- Task Loading ---
  async function loadTasks() {
    isLoading = true;
    error = null;
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
      tasks = {}; // Reset tasks on error
    }
  }

  // --- Task Selection / Adding ---
  function selectTask(taskName: string) {
    selectedTaskName = taskName;
    isAddingTask = false; // Ensure we are not in adding mode
    console.log("Selected task:", taskName);
  }

  function showNewTaskForm() {
    selectedTaskName = null; // Deselect any current task
    isAddingTask = true;
    console.log("Adding new task");
  }

  // --- Event Handlers from TaskDetail ---
  function handleTaskSaved(event: CustomEvent) {
    console.log("Task saved event received in App:", event.detail);
    // Reload tasks to reflect changes (simple approach)
    // A more optimized approach would be to update the specific task in the 'tasks' object
    loadTasks();
    // Optionally, keep the task selected or clear selection
    // if (event.detail.isNew) {
    //   selectedTaskName = event.detail.taskName; // Select the newly created task
    // }
  }

  function handleTaskDeleted(event: CustomEvent) {
    console.log("Task deleted event received in App:", event.detail);
    selectedTaskName = null; // Clear selection
    isAddingTask = false;
    loadTasks(); // Reload tasks
  }

  function handleNewTaskCreated(event: CustomEvent) {
    console.log("New task created event received in App:", event.detail);
    isAddingTask = false;
    loadTasks().then(() => {
      // Select the newly created task after reloading
      selectedTaskName = event.detail.taskName;
    });
  }

  // --- Lifecycle ---
  onMount(loadTasks);
</script>

<main class="app-container main-layout">
  <aside class="sidebar task-list-panel">
    <h2>Tasks</h2>
    {#if isLoading}
      <p>Loading tasks...</p>
    {:else if error}
      <p class="error">Error: {error}</p>
      <button on:click={loadTasks}>Retry</button>
    {:else}
      <ul class="task-list">
        {#each Object.keys(tasks) as taskName (taskName)}
          <li>
            <button class="task-button" class:active={taskName === selectedTaskName} class:new-task={tasks[taskName].isNew} class:modified-task={tasks[taskName].isModified} on:click={() => selectTask(taskName)}>
              {taskName}
            </button>
          </li>
        {/each}
        <li class="add-task-item">
          <button class="button task-button add-button" on:click={() => showNewTaskForm()}> + </button>
        </li>
      </ul>
    {/if}
  </aside>

  <section class="main-content task-detail-panel">
    {#if isAddingTask}
      <TaskDetail isNew={true} taskName="" taskConfig={{ interval: 10, downloaders: [], feeds: [], filter: null, extracter: null }} {apiFetch} on:taskSaved={handleNewTaskCreated} on:cancelAdd={() => (isAddingTask = false)} />
    {:else if selectedTaskName && tasks[selectedTaskName]}
      <TaskDetail isNew={false} taskName={selectedTaskName} taskConfig={tasks[selectedTaskName]} {apiFetch} on:taskSaved={handleTaskSaved} on:taskDeleted={handleTaskDeleted} />
    {:else if selectedTaskName}
      <p>Loading task details for {selectedTaskName}... or task data missing.</p>
    {:else}
      <p>Please select a task from the list or add a new one.</p>
    {/if}
  </section>
</main>
