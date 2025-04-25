<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import Modal from "./Modal.svelte"; // Import Modal

  // Define the structure for a downloader configuration
  interface DownloaderConfig {
    type: "aria2c" | "transmission"; // Add other types if needed
    host?: string;
    port?: number | null; // Optional, can be number or null
    rpcPath?: string; // Optional string
    token?: string;
    username?: string; // Optional string
    password?: string; // Optional string
    useHttps?: boolean; // Optional boolean
    autoCleanUp?: boolean;
  }

  export let downloaders: DownloaderConfig[] = []; // Use the interface
  export let apiFetch: (url: string, options?: RequestInit) => Promise<any>; // Function passed from parent

  const dispatch = createEventDispatcher();

  // --- SVG Icons (Consider moving to a dedicated utility or component) ---
  const EDIT_ICON_SVG = `
  <svg width='16px' height='16px' viewBox='0 0 24 24' fill='none' xmlns='http://www.w3.org/2000/svg'>
      <path d='M20,16v4a2,2,0,0,1-2,2H4a2,2,0,0,1-2-2V6A2,2,0,0,1,4,4H8' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' stroke-width='2'/>
      <polygon points='12.5 15.8 22 6.2 17.8 2 8.3 11.5 8 16 12.5 15.8' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' stroke-width='2'/>
  </svg>`;

  const DELETE_ICON_SVG = `
  <svg width='16px' height='16px' viewBox='0 0 24 24' fill='none' xmlns='http://www.w3.org/2000/svg'>
      <path d='M10 12V17 M14 12V17 M4 7H20 M6 10V18C6 19.66 7.34 21 9 21H15C16.66 21 18 19.66 18 18V10 M9 5C9 3.9 9.9 3 11 3H13C14.1 3 15 3.9 15 5V7H9V5Z' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'/>
  </svg>`;

  // --- Modal State ---
  let showDownloaderModal = false;
  let modalTitle = "";
  // Define a default structure using the interface
  const defaultDownloader: DownloaderConfig = {
    type: "aria2c",
    host: undefined,
    port: null,
    rpcPath: undefined,
    token: undefined,
    username: undefined,
    password: undefined,
    useHttps: false,
    autoCleanUp: false
  };
  let currentDownloaderData: DownloaderConfig = { ...defaultDownloader }; // Use the interface
  let editingDownloaderIndex: number | null = null;

  // --- Drag and Drop State ---
  let dragStartIndex: number | null = null;
  let dragOverIndex: number | null = null;

  // --- Helper Functions ---
  function getRpcUrl(downloader: any): string {
    const defaultPorts: { [key: string]: number } = {
      aria2c: 6800,
      transmission: 9091,
    };
    const defaultRpcPaths: { [key: string]: string } = {
      aria2c: "/jsonrpc",
      transmission: "/transmission/rpc",
    };

    const protocol = downloader.useHttps ? "https://" : "http://";
    const port = downloader.port || defaultPorts[downloader.type] || "";
    const rpcPath = downloader.rpcPath || defaultRpcPaths[downloader.type] || "";

    return `${protocol}${downloader.host || "localhost"}${port ? ":" + port : ""}${rpcPath}`;
  }

  // --- Event Handlers ---
  function openAddModal() {
    modalTitle = "Add New Downloader";
    // Reset to default, ensuring a fresh object copy
    currentDownloaderData = { ...defaultDownloader };
    editingDownloaderIndex = null;
    showDownloaderModal = true;
  }

  function openEditModal(index: number) {
    modalTitle = "Edit Downloader";
    // Make a deep copy to avoid modifying the original array directly during editing
    currentDownloaderData = JSON.parse(JSON.stringify(downloaders[index]));
    // Ensure all potential fields exist, even if undefined in original data
    currentDownloaderData = { ...defaultDownloader, ...currentDownloaderData };
    editingDownloaderIndex = index;
    showDownloaderModal = true;
  }

  function saveDownloader() {
    // Basic validation (can be expanded)
    if (!currentDownloaderData.type) {
      alert("Downloader type is required.");
      return;
    }

    // Clean up data according to the interface
    const dataToSave: DownloaderConfig = { ...currentDownloaderData };

    dataToSave.host = dataToSave.host?.trim() || undefined;

    // Handle port: Convert empty input or 0 to null, otherwise ensure it's a number
    const portInput = currentDownloaderData.port; // Get value from potentially bound input
    // Check for undefined, null, or 0
    if (portInput === undefined || portInput === null || Number(portInput) === 0) {
      dataToSave.port = null;
    } else {
      const parsedPort = Number(portInput);
      dataToSave.port = isNaN(parsedPort) ? null : parsedPort; // Assign null if parsing fails
    }

    // Handle optional strings: Convert empty strings to undefined
    dataToSave.token = dataToSave.token || undefined;
    dataToSave.username = dataToSave.username?.trim() || undefined;
    dataToSave.password = dataToSave.password|| undefined;
    dataToSave.rpcPath = dataToSave.rpcPath?.trim() || undefined;
    dataToSave.useHttps = dataToSave.useHttps || false;
    dataToSave.autoCleanUp = dataToSave.autoCleanUp || false;

    let updatedDownloaders: DownloaderConfig[]; // Use the interface
    if (editingDownloaderIndex !== null) {
      // Editing existing
      updatedDownloaders = [...downloaders];
      updatedDownloaders[editingDownloaderIndex] = dataToSave;
    } else {
      // Adding new
      updatedDownloaders = [...downloaders, dataToSave];
    }
    dispatch("update:downloaders", updatedDownloaders);
    showDownloaderModal = false; // Close modal
  }

  function handleDelete(index: number) {
    if (confirm(`Are you sure you want to delete downloader #${index + 1}?`)) {
      console.log("Delete downloader clicked for index:", index);
      // Create a new array without the deleted item
      const updatedDownloaders = downloaders.filter((_, i) => i !== index);
      // Notify the parent component of the change
      dispatch("update:downloaders", updatedDownloaders);
    }
  }

  // --- Drag and Drop Handlers ---
  function handleDragStart(event: DragEvent, index: number) {
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = "move";
      event.dataTransfer.setData("text/plain", index.toString()); // Store index
      dragStartIndex = index;
      (event.target as HTMLLIElement).classList.add("dragging");
    }
  }

  function handleDragOver(event: DragEvent, index: number) {
    event.preventDefault(); // Necessary to allow drop
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "move";
    }
    dragOverIndex = index; // Keep track of the element being dragged over
    // Add visual cue to the list item being hovered over
    const targetElement = event.currentTarget as HTMLLIElement;
    // Avoid adding class to the item being dragged
    if (dragStartIndex !== index) {
      targetElement.classList.add("drag-over");
    }
  }

  function handleDragLeave(event: DragEvent) {
    // Remove visual cue when dragging leaves the element
    (event.currentTarget as HTMLLIElement).classList.remove("drag-over");
    dragOverIndex = null;
  }

  function handleDrop(event: DragEvent, dropIndex: number) {
    event.preventDefault();
    (event.currentTarget as HTMLLIElement).classList.remove("drag-over"); // Clean up visual cue

    if (dragStartIndex === null || dragStartIndex === dropIndex) {
      dragStartIndex = null; // Reset if dropped on itself or invalid start
      return;
    }

    const draggedItem = downloaders[dragStartIndex];
    const remainingItems = downloaders.filter((_, i) => i !== dragStartIndex);

    // Insert the dragged item at the drop index
    const reorderedDownloaders = [...remainingItems.slice(0, dropIndex), draggedItem, ...remainingItems.slice(dropIndex)];

    dispatch("update:downloaders", reorderedDownloaders);
    dragStartIndex = null; // Reset state
  }

  function handleDragEnd(event: DragEvent) {
    // Clean up dragging class and reset state
    (event.target as HTMLLIElement).classList.remove("dragging");
    // Also remove any lingering drag-over classes if the drop happened outside a valid target
    document.querySelectorAll(".drag-over").forEach((el) => el.classList.remove("drag-over"));
    dragStartIndex = null;
    dragOverIndex = null;
  }
</script>

<!-- Downloader Add/Edit Modal -->
<Modal bind:showModal={showDownloaderModal} title={modalTitle} on:close={() => (showDownloaderModal = false)}>
  <form slot="body" class="modal-form" on:submit|preventDefault={saveDownloader}>
    <div class="form-group">
      <label for="downloader-type">Type</label>
      <select id="downloader-type" bind:value={currentDownloaderData.type} required>
        <option value="aria2c">Aria2c</option>
        <option value="transmission">Transmission</option>
        <!-- Add other types if supported -->
      </select>
    </div>
    <div class="form-group">
      <label for="downloader-host">Host (Optional)</label>
      <input type="text" id="downloader-host" bind:value={currentDownloaderData.host} placeholder="e.g., localhost or 192.168.1.10" required />
    </div>
    <div class="form-group">
      <label for="downloader-port">Port (Optional)</label>
      <input
        type="number"
        id="downloader-port"
        bind:value={currentDownloaderData.port}
        placeholder="e.g., 6800 (Aria2c), 9091 (Transmission)"
        min="1"
        max="65535"
      />
      <small>Leave blank for default port.</small>
    </div>
    <div class="form-group">
      <label for="downloader-rpcPath">RPC Path (Optional)</label>
      <input type="text" id="downloader-rpcPath" bind:value={currentDownloaderData.rpcPath} placeholder="e.g., /jsonrpc (Aria2c), /transmission/rpc" />
      <small>Leave blank for default path.</small>
    </div>
    <div class="form-group">
      <label for="downloader-token">Username (Optional)</label>
      <input type="text" id="downloader-token" bind:value={currentDownloaderData.token} />
    </div>
    <div class="form-group">
      <label for="downloader-username">Username (Optional)</label>
      <input type="text" id="downloader-username" bind:value={currentDownloaderData.username} />
    </div>
    <div class="form-group">
      <label for="downloader-password">Password (Optional)</label>
      <input type="password" id="downloader-password" bind:value={currentDownloaderData.password} />
    </div>
    <div class="form-group checkbox-group">
      <input type="checkbox" id="downloader-useHttps" bind:checked={currentDownloaderData.useHttps} />
      <label for="downloader-useHttps">Use HTTPS</label>
    </div>
    <div class="form-group checkbox-group">
      <input type="checkbox" id="downloader-autoCleanUp" bind:checked={currentDownloaderData.autoCleanUp} />
      <label for="downloader-autoCleanUp">Auto CleanUp</label>
    </div>
    <!-- Hidden submit button to allow Enter key submission -->
    <button type="submit" style="display: none;" aria-hidden="true"></button>
  </form>
  <div slot="footer">
    <button type="button" class="button secondary-button" on:click={() => (showDownloaderModal = false)}>Cancel</button>
    <button type="button" class="button primary-button" on:click={saveDownloader}>Save Downloader</button>
  </div>
</Modal>

<div class="list-section">
  <h4>Downloaders</h4>
  {#if downloaders && downloaders.length > 0}
    <ul class="list-items" id="downloader-list">
      {#each downloaders as downloader, index (index)}
        <li
          class="list-item draggable-item"
          data-index={index}
          draggable="true"
          on:dragstart={(e) => handleDragStart(e, index)}
          on:dragover={(e) => handleDragOver(e, index)}
          on:dragleave={handleDragLeave}
          on:drop={(e) => handleDrop(e, index)}
          on:dragend={handleDragEnd}
          class:drag-over={dragOverIndex === index && dragStartIndex !== index}
        >
          <span class="item-content">
            <span class="drag-handle">::</span>
            <strong>Type:</strong>
            {downloader.type} | <strong>RPC URL:</strong>
            {getRpcUrl(downloader)}
          </span>
          <div class="list-item-actions">
            <button type="button" class="button icon-button secondary-button" on:click={() => openEditModal(index)} title="Edit Downloader">
              {@html EDIT_ICON_SVG}
            </button>
            <button type="button" class="button icon-button danger-button" on:click={() => handleDelete(index)} title="Delete Downloader">
              {@html DELETE_ICON_SVG}
            </button>
          </div>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="empty-list-message">No downloaders configured.</p>
  {/if}

  <button type="button" class="button secondary-button add-item-button" on:click={openAddModal}> Add Downloader </button>
</div>
