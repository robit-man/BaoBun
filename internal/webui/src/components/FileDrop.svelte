<script lang="ts">
  import { createEventDispatcher, onMount, onDestroy } from "svelte";

  type DroppedBaoFile = {
    name: string;
    data: Uint8Array;
  };

  const dispatch = createEventDispatcher<{
    load: { files: DroppedBaoFile[] };
  }>();

  let isDragging = false;
  let error: string | null = null;
  let dragCounter = 0;

  function onDragEnter(e: DragEvent) {
    e.preventDefault();
    dragCounter++;
    isDragging = true;
  }

  function onDragLeave(e: DragEvent) {
    e.preventDefault();
    dragCounter--;
    if (dragCounter <= 0) {
      isDragging = false;
    }
  }

  function onDragOver(e: DragEvent) {
    e.preventDefault();
  }

  async function onDrop(e: DragEvent) {
    e.preventDefault();
    dragCounter = 0;
    isDragging = false;
    error = null;

    const dropped = Array.from(e.dataTransfer?.files ?? []);
    if (dropped.length === 0) return;

    try {
      const payload: DroppedBaoFile[] = [];
      for (const file of dropped) {
        const data = new Uint8Array(await file.arrayBuffer());
        payload.push({
          name: file.name,
          data,
        });
      }
      dispatch("load", { files: payload });
    } catch {
      error = "Failed to read dropped file(s).";
    }
  }

  // Prevent browser from opening the file
  onMount(() => {
    window.addEventListener("dragenter", onDragEnter);
    window.addEventListener("dragleave", onDragLeave);
    window.addEventListener("dragover", onDragOver);
    window.addEventListener("drop", onDrop);
  });

  onDestroy(() => {
    window.removeEventListener("dragenter", onDragEnter);
    window.removeEventListener("dragleave", onDragLeave);
    window.removeEventListener("dragover", onDragOver);
    window.removeEventListener("drop", onDrop);
  });
</script>

{#if isDragging}
  <div class="overlay">
    <div class="content">
      <h2>Drop files to import</h2>
      {#if error}
        <p class="error">{error}</p>
      {/if}
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 9999;
    pointer-events: none;
  }

  .content {
    background: #111;
    border: 2px dashed #c7aa81;
    padding: 3rem 4rem;
    border-radius: 12px;
    color: white;
    text-align: center;
  }

  .error {
    color: #ff6b6b;
    margin-top: 1rem;
  }
</style>
