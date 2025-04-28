<script lang="ts">
  interface Props {
    item: any;
    index: number;
    draggable?: boolean;
    isDraggedOver?: boolean;
    dragStart?: any;
    dragEnd?: any;
    dragOver?: any;
    dragLeave?: any;
    drop?: any;
    edit?: any;
    del?: any;
    children?: import("svelte").Snippet;
  }

  let { item, index, draggable = false, isDraggedOver = false, dragStart, dragEnd, dragOver, dragLeave, drop, edit, del, children }: Props = $props();

  // Shared SVG Icons
  export const EDIT_ICON_SVG = `
  <svg width='16px' height='16px' viewBox='0 0 24 24' fill='none' xmlns='http://www.w3.org/2000/svg'>
      <path d='M20,16v4a2,2,0,0,1-2,2H4a2,2,0,0,1-2-2V6A2,2,0,0,1,4,4H8' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' stroke-width='2'/>
      <polygon points='12.5 15.8 22 6.2 17.8 2 8.3 11.5 8 16 12.5 15.8' stroke='currentColor' stroke-linecap='round' stroke-linejoin='round' stroke-width='2'/>
  </svg>`;

  export const DELETE_ICON_SVG = `
  <svg width='16px' height='16px' viewBox='0 0 24 24' fill='none' xmlns='http://www.w3.org/2000/svg'>
      <path d='M10 12V17 M14 12V17 M4 7H20 M6 10V18C6 19.66 7.34 21 9 21H15C16.66 21 18 19.66 18 18V10 M9 5C9 3.9 9.9 3 11 3H13C14.1 3 15 3.9 15 5V7H9V5Z' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'/>
  </svg>`;

  // Drag and Drop Handlers
  function handleDragStart(event: DragEvent) {
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = "move";
      event.dataTransfer.setData("text/plain", index.toString());
      dragStart(index);
      (event.target as HTMLElement).classList.add("dragging");
    }
  }

  function handleDragOver(event: DragEvent) {
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "move";
    }
    dragOver(index);
  }

  function handleDragLeave(event: DragEvent) {
    dragLeave(index);
  }

  function handleDrop(event: DragEvent) {
    event.preventDefault();
    drop(index);
  }

  function handleDragEnd(event: DragEvent) {
    (event.target as HTMLElement).classList.remove("dragging");
    dragEnd();
  }

  // Action Handlers
  function handleEdit() {
    edit(index);
  }

  function handleDelete() {
    del(index);
  }
</script>

<li
  class="list-item {draggable ? 'draggable-item' : ''}"
  data-index={index}
  {draggable}
  ondragstart={handleDragStart}
  ondragover={handleDragOver}
  ondragleave={handleDragLeave}
  ondrop={handleDrop}
  ondragend={handleDragEnd}
  class:drag-over={draggable && isDraggedOver}
>
  <span class="item-content">
    {#if draggable}
      <span class="drag-handle">::</span>
    {/if}
    {#if children}{@render children()}{:else}{item}{/if}
  </span>

  <div class="list-item-actions">
    <button type="button" class="button icon-button secondary-button" onclick={handleEdit} title="Edit">
      {@html EDIT_ICON_SVG}
    </button>
    <button type="button" class="button icon-button danger-button" onclick={handleDelete} title="Delete">
      {@html DELETE_ICON_SVG}
    </button>
  </div>
</li>

<style>
  .list-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 10px;
    margin-bottom: 5px;
    background-color: #f8f9fa;
    border: 1px solid #eee;
    border-radius: 4px;
    word-break: break-all;
  }

  .list-item:last-child {
    border-bottom: none;
  }

  .item-content {
    flex-grow: 1;
    margin-right: 1rem;
  }

  .list-item-actions {
    display: flex;
  }

  .icon-button {
    margin-left: 8px;
    padding: 3px 3px;
    font-size: 0.85em;
    line-height: 1;
  }

  .drag-handle {
    cursor: grab;
    margin-right: 0.5rem;
    color: #aaa;
    display: inline-block;
    user-select: none;
  }

  .draggable-item {
    cursor: move;
  }

  .draggable-item:active .drag-handle {
    cursor: grabbing;
  }

  .drag-over {
    border-top: 2px solid #007bff;
  }
</style>
