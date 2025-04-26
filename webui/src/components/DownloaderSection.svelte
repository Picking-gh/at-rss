<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import Modal from "./Modal.svelte";

  // Define the structure for a downloader configuration
  interface DownloaderConfig {
    type: "aria2c" | "transmission";
    host?: string;
    port?: number | null;
    rpcPath?: string;
    token?: string;
    username?: string;
    password?: string;
    useHttps?: boolean;
    autoCleanUp?: boolean;
  }

  export let downloaders: DownloaderConfig[] = []; // Use the interface

  const dispatch = createEventDispatcher();

  import ListItem from "./ListItem.svelte";

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
    autoCleanUp: false,
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
    currentDownloaderData = structuredClone(downloaders[index]);
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
    dataToSave.password = dataToSave.password || undefined;
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
    showDownloaderModal = false;
  }

  function handleDelete(index: number) {
    if (confirm(`Are you sure you want to delete downloader #${index + 1}?`)) {
      // Create a new array without the deleted item
      const updatedDownloaders = downloaders.filter((_, i) => i !== index);
      // Notify the parent component of the change
      dispatch("update:downloaders", updatedDownloaders);
    }
  }

  // --- Drag and Drop Handlers ---
  function handleDragStart(event: CustomEvent) {
    dragStartIndex = event.detail.index;
  }

  function handleDragOver(event: CustomEvent) {
    dragOverIndex = event.detail.index;
  }

  function handleDragLeave() {
    dragOverIndex = null;
  }

  function handleDrop(event: CustomEvent) {
    const dropIndex = event.detail.index;

    if (dragStartIndex === null || dragStartIndex === dropIndex) {
      dragStartIndex = null;
      return;
    }

    const draggedItem = downloaders[dragStartIndex];
    const remainingItems = downloaders.filter((_, i) => i !== dragStartIndex);
    const reorderedDownloaders = [...remainingItems.slice(0, dropIndex), draggedItem, ...remainingItems.slice(dropIndex)];

    dispatch("update:downloaders", reorderedDownloaders);
    dragStartIndex = null;
  }

  function handleDragEnd() {
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
        placeholder={currentDownloaderData.type === "aria2c" ? "e.g., 6800" : "e.g., 9091"}
        min="1"
        max="65535"
      />
    </div>
    <div class="form-group">
      <label for="downloader-rpcPath">RPC Path (Optional)</label>
      <input
        type="text"
        id="downloader-rpcPath"
        bind:value={currentDownloaderData.rpcPath}
        placeholder={currentDownloaderData.type === "aria2c" ? "e.g., /jsonrpc" : "e.g., /transmission/rpc"}
      />
    </div>
    {#if currentDownloaderData.type === "aria2c"}
      <div class="form-group">
        <label for="downloader-token">Token (Optional)</label>
        <input type="text" id="downloader-token" bind:value={currentDownloaderData.token} placeholder="Aria2c RPC secret token" />
      </div>
    {:else}
      <div class="form-group">
        <label for="downloader-username">Username (Optional)</label>
        <input type="text" id="downloader-username" bind:value={currentDownloaderData.username} />
      </div>
      <div class="form-group">
        <label for="downloader-password">Password (Optional)</label>
        <input type="password" id="downloader-password" bind:value={currentDownloaderData.password} />
      </div>
    {/if}
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
    <button type="button" class="button primary-button" on:click={saveDownloader}>Save</button>
    <button type="button" class="button secondary-button" on:click={() => (showDownloaderModal = false)}>Cancel</button>
  </div>
</Modal>

<div class="form-section">
  <h3>Downloaders</h3>
  <div class="list-section">
    {#if downloaders && downloaders.length > 0}
      <ul class="list-items" id="downloader-list">
        {#each downloaders as downloader, index (index)}
          <ListItem
            item={downloader}
            {index}
            draggable={true}
            bind:dragStartIndex
            bind:dragOverIndex
            on:dragstart={handleDragStart}
            on:dragover={handleDragOver}
            on:dragleave={handleDragLeave}
            on:drop={handleDrop}
            on:dragend={handleDragEnd}
            on:edit={() => openEditModal(index)}
            on:delete={() => handleDelete(index)}
          >
            <strong>Type:</strong>
            {downloader.type} | <strong>RPC URL:</strong>
            {getRpcUrl(downloader)}
          </ListItem>
        {/each}
      </ul>
    {:else}
      <p class="empty-list-message">No downloaders configured.</p>
    {/if}

    <button type="button" class="button secondary-button add-item-button" on:click={openAddModal}> Add Downloader </button>
  </div>
</div>
