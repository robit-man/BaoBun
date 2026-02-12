<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte";
  import {
    applyTorrentAction,
    fetchHiddenCount,
    unhideTorrents,
  } from "./api";
  import type { TorrentStatus } from "./types";
  import baoSwarmIcon from "../assets/baoswarm.png";

  type UploadFile = { name: string; data: Uint8Array };
  type AddMode = "magnet" | "file";
  type FilterId =
    | "all"
    | "downloading"
    | "seeding"
    | "completed"
    | "stopped"
    | "active";

  export let torrents: TorrentStatus[] = [];
  export let error: string | null = null;
  export let uploadError: string | null = null;
  export let uploadMessage: string | null = null;

  const dispatch = createEventDispatcher<{
    load: { files: UploadFile[] };
    openConfig: void;
    refresh: void;
  }>();

  let activeFilter: FilterId = "all";
  let searchTerm = "";
  let selectedIds = new Set<string>();
  let selectAllEl: HTMLInputElement | null = null;
  let filePickerEl: HTMLInputElement | null = null;
  let addMode: AddMode = "file";
  let addModalOpen = false;
  let addDragActive = false;
  let addFiles: File[] = [];
  let addSource = "";
  let addCategory = "None";
  let addSavePath = "Default";
  let addStartBao = true;
  let addSequential = false;
  let addBusy = false;
  let addError: string | null = null;

  let actionBusy = false;
  let actionError: string | null = null;
  let actionMessage: string | null = null;

  let hiddenCount = 0;
  let hideModalOpen = false;
  let unhideModalOpen = false;
  let hidePasskey = "";
  let unhidePasskey = "";

  const filterDefs: { id: FilterId; label: string; icon: string }[] = [
    { id: "all", label: "All", icon: "\u25A6" },
    { id: "downloading", label: "Downloading", icon: "\u2193" },
    { id: "seeding", label: "Seeding", icon: "\u2191" },
    { id: "completed", label: "Completed", icon: "\u25EF" },
    { id: "stopped", label: "Stopped", icon: "\u25A0" },
    { id: "active", label: "Active", icon: "\u26A1" },
  ];

  $: filtered = torrents
    .filter((torrent) => matchesFilter(torrent, activeFilter))
    .filter((torrent) =>
      torrent.name.toLowerCase().includes(searchTerm.trim().toLowerCase())
    )
    .sort((a, b) => a.name.localeCompare(b.name));

  $: totalDownRate = torrents.reduce((sum, torrent) => sum + torrent.downRate, 0);
  $: totalUpRate = torrents.reduce((sum, torrent) => sum + torrent.upRate, 0);

  $: allVisibleSelected =
    filtered.length > 0 && filtered.every((torrent) => selectedIds.has(torrent.id));
  $: someVisibleSelected =
    filtered.length > 0 && filtered.some((torrent) => selectedIds.has(torrent.id));
  $: selectedCount = selectedIds.size;

  $: if (selectAllEl) {
    selectAllEl.indeterminate = !allVisibleSelected && someVisibleSelected;
  }

  onMount(() => {
    refreshHiddenCount();
  });

  function selectedIDs(): string[] {
    return Array.from(selectedIds);
  }

  function clearSelection() {
    selectedIds = new Set<string>();
  }

  async function refreshHiddenCount() {
    try {
      hiddenCount = await fetchHiddenCount();
    } catch {
      hiddenCount = 0;
    }
  }

  async function runAction(action: "pause" | "archive" | "delete") {
    if (selectedCount === 0 || actionBusy) {
      return;
    }

    actionBusy = true;
    actionError = null;
    actionMessage = null;

    try {
      const result = await applyTorrentAction(action, selectedIDs());
      hiddenCount = result.hidden;
      actionMessage = result.message;
      clearSelection();
      dispatch("refresh");
      await refreshHiddenCount();
    } catch (err) {
      actionError =
        err instanceof Error ? err.message : `Failed to ${action} selected torrents`;
    } finally {
      actionBusy = false;
    }
  }

  function openHideModal() {
    if (selectedCount === 0 || actionBusy) {
      return;
    }
    hidePasskey = "";
    actionError = null;
    hideModalOpen = true;
  }

  async function confirmHide() {
    if (selectedCount === 0 || actionBusy) {
      return;
    }
    if (!hidePasskey.trim()) {
      actionError = "Passkey is required to hide selected items.";
      return;
    }

    actionBusy = true;
    actionError = null;
    actionMessage = null;
    try {
      const result = await applyTorrentAction("hide", selectedIDs(), hidePasskey);
      hiddenCount = result.hidden;
      actionMessage = result.message;
      hideModalOpen = false;
      clearSelection();
      dispatch("refresh");
      await refreshHiddenCount();
    } catch (err) {
      actionError =
        err instanceof Error ? err.message : "Failed to hide selected torrents";
    } finally {
      actionBusy = false;
    }
  }

  function openUnhideModal() {
    if (actionBusy) {
      return;
    }
    unhidePasskey = "";
    actionError = null;
    unhideModalOpen = true;
  }

  async function confirmUnhide() {
    if (actionBusy) {
      return;
    }
    if (!unhidePasskey.trim()) {
      actionError = "Passkey is required to unhide items.";
      return;
    }

    actionBusy = true;
    actionError = null;
    actionMessage = null;
    try {
      const result = await unhideTorrents(unhidePasskey);
      hiddenCount = result.hidden;
      actionMessage = result.message;
      unhideModalOpen = false;
      dispatch("refresh");
      await refreshHiddenCount();
    } catch (err) {
      actionError = err instanceof Error ? err.message : "Failed to unhide torrents";
    } finally {
      actionBusy = false;
    }
  }

  function isDownloading(torrent: TorrentStatus): boolean {
    return torrent.state === "downloading" && torrent.remaining > 0;
  }

  function isSeeding(torrent: TorrentStatus): boolean {
    return torrent.state === "seeding" || (torrent.remaining === 0 && torrent.upRate > 0);
  }

  function isCompleted(torrent: TorrentStatus): boolean {
    return torrent.remaining === 0;
  }

  function isStopped(torrent: TorrentStatus): boolean {
    return !isDownloading(torrent) && !isSeeding(torrent);
  }

  function isActive(torrent: TorrentStatus): boolean {
    return torrent.downRate > 0 || torrent.upRate > 0;
  }

  function matchesFilter(torrent: TorrentStatus, filterId: FilterId): boolean {
    switch (filterId) {
      case "downloading":
        return isDownloading(torrent);
      case "seeding":
        return isSeeding(torrent);
      case "completed":
        return isCompleted(torrent);
      case "stopped":
        return isStopped(torrent);
      case "active":
        return isActive(torrent);
      default:
        return true;
    }
  }

  function filterCount(filterId: FilterId): number {
    if (filterId === "all") {
      return torrents.length;
    }
    return torrents.filter((torrent) => matchesFilter(torrent, filterId)).length;
  }

  function formatBytes(value: number): string {
    if (value < 1024) return `${value} B`;
    if (value < 1024 ** 2) return `${(value / 1024).toFixed(1)} KB`;
    if (value < 1024 ** 3) return `${(value / 1024 ** 2).toFixed(2)} MB`;
    return `${(value / 1024 ** 3).toFixed(2)} GB`;
  }

  function formatRate(value: number): string {
    if (value < 1024) return `${value} B/s`;
    if (value < 1024 ** 2) return `${(value / 1024).toFixed(2)} KB/s`;
    if (value < 1024 ** 3) return `${(value / 1024 ** 2).toFixed(2)} MB/s`;
    return `${(value / 1024 ** 3).toFixed(2)} GB/s`;
  }

  function formatRatio(value: number): string {
    return value.toFixed(2);
  }

  function progressPercent(torrent: TorrentStatus): number {
    if (torrent.fileSize <= 0) return 0;
    const pct = ((torrent.fileSize - torrent.remaining) / torrent.fileSize) * 100;
    return Math.max(0, Math.min(100, Math.round(pct)));
  }

  function progressEta(torrent: TorrentStatus): string {
    if (torrent.remaining <= 0) return "Done";
    if (torrent.downRate <= 0) return "--";

    const seconds = Math.ceil(torrent.remaining / torrent.downRate);
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.ceil(seconds / 60)}m`;

    const hours = Math.floor(seconds / 3600);
    const mins = Math.ceil((seconds % 3600) / 60);
    if (hours < 24) return `${hours}h ${mins}m`;

    const days = Math.floor(hours / 24);
    const hourPart = hours % 24;
    return `${days}d ${hourPart}h`;
  }

  function statusLabel(
    torrent: TorrentStatus
  ): "Downloading" | "Seeding" | "Paused" | "Stopped" {
    if (torrent.state === "paused") {
      return "Paused";
    }
    if (isDownloading(torrent)) {
      return "Downloading";
    }
    if (isSeeding(torrent)) {
      return "Seeding";
    }
    return "Stopped";
  }

  function statusClass(torrent: TorrentStatus): string {
    const label = statusLabel(torrent).toLowerCase();
    return `status-pill ${label}`;
  }

  function speedDisplay(value: number): string {
    if (value <= 0) {
      return "--";
    }
    return formatRate(value);
  }

  function isSelected(id: string): boolean {
    return selectedIds.has(id);
  }

  function toggleRow(id: string, checked: boolean): void {
    const next = new Set(selectedIds);
    if (checked) {
      next.add(id);
    } else {
      next.delete(id);
    }
    selectedIds = next;
  }

  function toggleAll(checked: boolean): void {
    const next = new Set(selectedIds);
    for (const torrent of filtered) {
      if (checked) {
        next.add(torrent.id);
      } else {
        next.delete(torrent.id);
      }
    }
    selectedIds = next;
  }

  function openAddModal() {
    addMode = "file";
    addModalOpen = true;
    addDragActive = false;
    addFiles = [];
    addSource = "";
    addCategory = "None";
    addSavePath = "Default";
    addStartBao = true;
    addSequential = false;
    addError = null;
  }

  function closeAddModal() {
    if (addBusy) {
      return;
    }
    addModalOpen = false;
    addDragActive = false;
    addError = null;
  }

  function chooseAddMode(mode: AddMode) {
    addMode = mode;
    addError = null;
  }

  function triggerAddPicker() {
    if (addMode !== "file") {
      return;
    }
    if (filePickerEl) {
      filePickerEl.value = "";
      filePickerEl.click();
    }
  }

  function ingestFiles(files: File[]) {
    if (files.length === 0) {
      return;
    }

    const next = [...addFiles];
    const existingKeys = new Set(
      next.map((file) => `${file.name}:${file.size}:${file.lastModified}`)
    );

    for (const file of files) {
      const key = `${file.name}:${file.size}:${file.lastModified}`;
      if (existingKeys.has(key)) {
        continue;
      }
      existingKeys.add(key);
      next.push(file);
    }

    addFiles = next;
    addError = null;
  }

  function onPickerChange(event: Event) {
    const input = event.currentTarget as HTMLInputElement | null;
    const files = Array.from(input?.files ?? []);
    ingestFiles(files);

    if (input) {
      input.value = "";
    }
  }

  function onAddDragEnter(event: DragEvent) {
    event.stopPropagation();
    addDragActive = true;
  }

  function onAddDragOver(event: DragEvent) {
    event.stopPropagation();
    addDragActive = true;
  }

  function onAddDragLeave(event: DragEvent) {
    event.stopPropagation();
    addDragActive = false;
  }

  function onAddDrop(event: DragEvent) {
    event.stopPropagation();
    addDragActive = false;
    ingestFiles(Array.from(event.dataTransfer?.files ?? []));
  }

  function selectedFilesLabel(): string {
    if (addFiles.length === 0) {
      return "";
    }

    const preview = addFiles.slice(0, 2).map((file) => file.name).join(", ");
    const extra = addFiles.length > 2 ? ` +${addFiles.length - 2} more` : "";
    return `${addFiles.length} file(s): ${preview}${extra}`;
  }

  function inferRemoteFileName(source: string, contentDisposition: string | null): string {
    if (contentDisposition) {
      const cdMatch =
        /filename\*=UTF-8''([^;]+)|filename="?([^\";]+)"?/i.exec(contentDisposition);
      const raw = cdMatch?.[1] ?? cdMatch?.[2] ?? "";
      if (raw) {
        try {
          return decodeURIComponent(raw);
        } catch {
          return raw;
        }
      }
    }

    try {
      const parsed = new URL(source);
      const fileName = decodeURIComponent(parsed.pathname.split("/").pop() ?? "");
      if (fileName) {
        return fileName;
      }
    } catch {
      // Keep fallback below for invalid URL formatting.
    }

    return "remote-upload.bin";
  }

  async function submitAddModal() {
    if (addBusy) {
      return;
    }

    addError = null;
    addBusy = true;

    try {
      if (addMode === "magnet") {
        const source = addSource.trim();
        if (!source) {
          addError = "Enter a magnet or URL.";
          return;
        }
        if (source.startsWith("magnet:")) {
          addError = "Magnet links are not supported by the backend yet. Use Bao File mode.";
          return;
        }

        const response = await fetch(source);
        if (!response.ok) {
          addError = `URL fetch failed (${response.status}).`;
          return;
        }

        const name = inferRemoteFileName(
          source,
          response.headers.get("content-disposition")
        );

        dispatch("load", {
          files: [{ name, data: new Uint8Array(await response.arrayBuffer()) }],
        });
        addModalOpen = false;
        return;
      }

      if (addFiles.length === 0) {
        addError = "Select at least one file to add.";
        return;
      }

      const payload: UploadFile[] = [];
      for (const file of addFiles) {
        payload.push({
          name: file.name,
          data: new Uint8Array(await file.arrayBuffer()),
        });
      }

      dispatch("load", { files: payload });
      addModalOpen = false;
    } catch (err) {
      addError =
        err instanceof Error
          ? err.message
          : "Failed to prepare files for upload.";
    } finally {
      addBusy = false;
    }
  }

  function closeOnBackdropKeydown(
    event: KeyboardEvent,
    close: () => void
  ): void {
    if (event.key === "Escape" || event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      close();
    }
  }
</script>

<div class="surface">
  <div class="topbar">
    <div class="brand">
      <div class="brand-icon">
        <img src={baoSwarmIcon} alt="BaoSwarm" />
      </div>
      <div class="brand-title">BaoSwarm</div>
    </div>
    <button class="version-pill" type="button" on:click={() => dispatch("openConfig")}>
      Config
    </button>
  </div>

  <div class="controls-row">
    <div class="control-cluster">
      <button class="add-btn" type="button" on:click={openAddModal}>
        <span class="plus">+</span>
        <span>Add</span>
      </button>

      <div class="cluster-divider"></div>

      {#each filterDefs as filter}
        <button
          class="filter-pill {activeFilter === filter.id ? 'active' : ''}"
          type="button"
          on:click={() => (activeFilter = filter.id)}
        >
          <span class="filter-icon">{filter.icon}</span>
          <span>{filter.label}</span>
          {#if filter.id === "all"}
            <span class="count-pill">{filterCount(filter.id)}</span>
          {/if}
        </button>
      {/each}
    </div>

    <label class="search">
      <span class="search-icon">&#8981;</span>
      <input type="text" placeholder="Search torrents..." bind:value={searchTerm} />
    </label>
  </div>

  <input
    bind:this={filePickerEl}
    class="hidden-file-input"
    type="file"
    multiple
    accept="*/*"
    on:change={onPickerChange}
  />

  {#if error}
    <p class="notice error">{error}</p>
  {/if}
  {#if uploadError}
    <p class="notice error">{uploadError}</p>
  {/if}
  {#if uploadMessage}
    <p class="notice ok">{uploadMessage}</p>
  {/if}
  {#if actionError}
    <p class="notice error">{actionError}</p>
  {/if}
  {#if actionMessage}
    <p class="notice ok">{actionMessage}</p>
  {/if}

  <div class="table-shell">
    <div class="table-header grid-row">
      <div class="th select-col">
        <input
          bind:this={selectAllEl}
          class="torrent-select-toggle"
          type="checkbox"
          checked={allVisibleSelected}
          on:change={(event) => toggleAll((event.currentTarget as HTMLInputElement).checked)}
        />
      </div>
      <div class="th sortable active">NAME <span class="caret">&#9662;</span></div>
      <div class="th sortable">SIZE <span class="caret">&#9662;</span></div>
      <div class="th sortable">PROGRESS <span class="caret">&#9662;</span></div>
      <div class="th">STATUS</div>
      <div class="th sortable">DOWN <span class="caret">&#9662;</span></div>
      <div class="th sortable">UP <span class="caret">&#9662;</span></div>
      <div class="th speed-header down"><span class="speed-arrow">&#8595;</span><span>SPEED</span></div>
      <div class="th speed-header up"><span class="speed-arrow">&#8593;</span><span>SPEED</span></div>
      <div class="th sortable">RATIO <span class="caret">&#9662;</span></div>
      <div class="th actions-col"></div>
    </div>

    <div class="table-body">
      {#each filtered as torrent (torrent.id)}
        <div class="table-row grid-row">
          <div class="cell select-col">
            <input
              class="torrent-select-toggle"
              type="checkbox"
              checked={isSelected(torrent.id)}
              on:change={(event) =>
                toggleRow(torrent.id, (event.currentTarget as HTMLInputElement).checked)}
            />
          </div>

          <div class="cell name">{torrent.name}</div>
          <div class="cell size">{formatBytes(torrent.fileSize)}</div>

          <div class="cell progress-cell">
            {#if torrent.remaining === 0}
              <div class="progress-complete">
                <span class="complete-icon">&#9679;</span>
                <span>Complete</span>
              </div>
            {:else}
              <div class="progress-active">
                <div class="bar-track">
                  <div class="bar-fill" style="width: {progressPercent(torrent)}%"></div>
                </div>
                <div class="progress-meta">
                  <span>{progressEta(torrent)}</span>
                  <span>{progressPercent(torrent)}%</span>
                </div>
              </div>
            {/if}
          </div>

          <div class="cell">
            <span class={statusClass(torrent)}>{statusLabel(torrent)}</span>
          </div>

          <div class="cell numeric">{formatBytes(torrent.downloaded)}</div>
          <div class="cell numeric">{formatBytes(torrent.uploaded)}</div>
          <div class="cell numeric speed-down">{speedDisplay(torrent.downRate)}</div>
          <div class="cell numeric speed-up">{speedDisplay(torrent.upRate)}</div>
          <div class="cell numeric">{formatRatio(torrent.ratio)}</div>

          <div class="cell actions-col">
            <button class="row-action" type="button" aria-label="Open row actions">
              &rsaquo;
            </button>
          </div>
        </div>
      {/each}

      {#if filtered.length === 0}
        <div class="empty-state">No torrents match this filter.</div>
      {/if}
    </div>
  </div>

  <div class="status-bar">
    <div class="status-left">
      <span class="connected"><span class="dot"></span>Connected</span>
      <span class="agg down"><span>&#8595;</span>{formatRate(totalDownRate)}</span>
      <span class="agg up"><span>&#8593;</span>{formatRate(totalUpRate)}</span>
    </div>

    <div class="status-right">
      {#if selectedCount > 0}
        <div class="selection-actions">
          <button type="button" class="action pause" on:click={() => runAction("pause")} disabled={actionBusy}>
            Pause
          </button>
          <button type="button" class="action archive" on:click={() => runAction("archive")} disabled={actionBusy}>
            Archive
          </button>
          <button type="button" class="action delete" on:click={() => runAction("delete")} disabled={actionBusy}>
            Delete
          </button>
          <button type="button" class="action hide" on:click={openHideModal} disabled={actionBusy}>
            Hide
          </button>
        </div>
      {/if}

      <button type="button" class="hidden-btn" on:click={openUnhideModal} disabled={actionBusy}>
        Hidden
        {#if hiddenCount > 0}
          <span class="hidden-count">{hiddenCount}</span>
        {/if}
      </button>
    </div>
  </div>
</div>

{#if addModalOpen}
  <div
    class="add-modal-backdrop"
    role="button"
    tabindex="0"
    on:click|self={closeAddModal}
    on:keydown={(event) => closeOnBackdropKeydown(event, closeAddModal)}
  >
    <div class="add-modal" role="dialog" aria-modal="true" aria-labelledby="add-bao-title">
      <div class="add-modal-header">
        <div class="add-modal-title-wrap">
          <span class="add-modal-badge">+</span>
          <h2 id="add-bao-title">Add Bao</h2>
        </div>
        <button type="button" class="add-modal-close" on:click={closeAddModal} disabled={addBusy}>
          &#10005;
        </button>
      </div>

      <div class="add-modal-divider"></div>

      <div class="add-modal-body">
        <div class="add-mode-track">
          <button
            type="button"
            class="add-mode-segment {addMode === 'magnet' ? 'active' : ''}"
            on:click={() => chooseAddMode("magnet")}
            disabled={addBusy}
          >
            Magnet / URL
          </button>
          <button
            type="button"
            class="add-mode-segment {addMode === 'file' ? 'active' : ''}"
            on:click={() => chooseAddMode("file")}
            disabled={addBusy}
          >
            Bao File
          </button>
        </div>

        {#if addMode === "file"}
          <p class="add-field-label">Bao file</p>
          <button
            type="button"
            class="add-dropzone {addDragActive ? 'drag-active' : ''}"
            on:click={triggerAddPicker}
            on:dragenter|preventDefault={onAddDragEnter}
            on:dragover|preventDefault={onAddDragOver}
            on:dragleave|preventDefault={onAddDragLeave}
            on:drop|preventDefault={onAddDrop}
            disabled={addBusy}
          >
            <span class="add-drop-icon">&#x21E7;</span>
            <span class="add-drop-title">Click or drop .bao file</span>
            <span class="add-drop-subtitle">Any file type is accepted and auto-handled.</span>
          </button>
          {#if addFiles.length > 0}
            <p class="add-selected-files">{selectedFilesLabel()}</p>
          {/if}
        {:else}
          <label class="add-field-label" for="add-source">Magnet / URL</label>
          <input
            id="add-source"
            class="add-input"
            type="text"
            bind:value={addSource}
            placeholder="magnet:?xt=... or https://example.com/file"
            disabled={addBusy}
          />
          <p class="add-help-text">
            URL uploads are fetched and added. Magnet links are shown but not yet supported by the backend.
          </p>
        {/if}

        <div class="add-meta-grid">
          <label class="add-meta-field">
            <span>Category</span>
            <select bind:value={addCategory} disabled={addBusy}>
              <option value="None">None</option>
            </select>
          </label>
          <label class="add-meta-field">
            <span>Save path</span>
            <select bind:value={addSavePath} disabled={addBusy}>
              <option value="Default">Default</option>
            </select>
          </label>
        </div>

        <div class="add-options-row">
          <label class="add-checkbox">
            <input type="checkbox" bind:checked={addStartBao} disabled={addBusy} />
            <span>Start bao</span>
          </label>
          <label class="add-checkbox">
            <input type="checkbox" bind:checked={addSequential} disabled={addBusy} />
            <span>Sequential</span>
          </label>
        </div>

        {#if addError}
          <p class="add-error">{addError}</p>
        {/if}
      </div>

      <div class="add-modal-footer">
        <button type="button" class="add-footer-btn secondary" on:click={closeAddModal} disabled={addBusy}>
          Cancel
        </button>
        <button type="button" class="add-footer-btn primary" on:click={submitAddModal} disabled={addBusy}>
          Add Bao
        </button>
      </div>
    </div>
  </div>
{/if}

{#if hideModalOpen}
  <div
    class="modal-backdrop"
    role="button"
    tabindex="0"
    on:click|self={() => (hideModalOpen = false)}
    on:keydown={(event) =>
      closeOnBackdropKeydown(event, () => (hideModalOpen = false))}
  >
    <div class="modal-panel">
      <h3>Hide Selected</h3>
      <p>
        Enter a passkey to encrypt and hide {selectedCount} selected item(s). You will need
        the same passkey to unhide them.
      </p>
      <input
        type="password"
        bind:value={hidePasskey}
        placeholder="Passkey"
        autocomplete="off"
      />
      <div class="modal-actions">
        <button type="button" on:click={() => (hideModalOpen = false)} disabled={actionBusy}>
          Cancel
        </button>
        <button type="button" class="confirm" on:click={confirmHide} disabled={actionBusy}>
          Encrypt + Hide
        </button>
      </div>
    </div>
  </div>
{/if}

{#if unhideModalOpen}
  <div
    class="modal-backdrop"
    role="button"
    tabindex="0"
    on:click|self={() => (unhideModalOpen = false)}
    on:keydown={(event) =>
      closeOnBackdropKeydown(event, () => (unhideModalOpen = false))}
  >
    <div class="modal-panel">
      <h3>Unhide Torrents</h3>
      <p>Enter the passkey used when hiding the torrents.</p>
      <input
        type="password"
        bind:value={unhidePasskey}
        placeholder="Passkey"
        autocomplete="off"
      />
      <div class="modal-actions">
        <button type="button" on:click={() => (unhideModalOpen = false)} disabled={actionBusy}>
          Cancel
        </button>
        <button type="button" class="confirm" on:click={confirmUnhide} disabled={actionBusy}>
          Unhide
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .surface {
    min-height: calc(100vh - 28px);
    background: #141821;
    border: 1px solid #222a3b;
    border-radius: 14px;
    display: flex;
    flex-direction: column;
    color: #e4e8f1;
    box-shadow: 0 24px 44px rgba(0, 0, 0, 0.4);
    overflow: hidden;
  }

  .topbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 16px;
    border-bottom: 1px solid #222a3b;
    background: #10151f;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .brand-icon {
    width: 30px;
    height: 30px;
    border-radius: 8px;
    overflow: hidden;
  }

  .brand-icon img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .brand-title {
    font-size: 17px;
    font-weight: 600;
    letter-spacing: 0.01em;
  }

  .version-pill {
    border: 1px solid #2c3448;
    background: #161d2a;
    color: #c2cad7;
    padding: 6px 10px;
    border-radius: 999px;
    font-size: 12px;
    font-weight: 600;
  }

  .controls-row {
    display: flex;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 14px;
    border-bottom: 1px solid #20283a;
    background: #121825;
  }

  .control-cluster {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px;
    border: 1px solid #2a3348;
    border-radius: 1.5rem;
    background: #0f1521;
    overflow-x: auto;
  }

  .add-btn {
    border: 1px solid #2f3951;
    background: #1a2232;
    color: #e8edf8;
    border-radius: 999px;
    padding: 7px 12px;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    white-space: nowrap;
  }

  .plus {
    font-size: 16px;
    line-height: 1;
    color: #ffae00;
  }

  .cluster-divider {
    width: 1px;
    height: 24px;
    background: #263149;
  }

  .filter-pill {
    border: 1px solid transparent;
    background: transparent;
    color: #a7b0be;
    border-radius: 999px;
    padding: 6px 10px;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    white-space: nowrap;
    font-size: 12px;
  }

  .filter-pill.active {
    background: #4f402a;
    border-color: #b99765;
    color: #f9e8d2;
  }

  .filter-icon {
    width: 14px;
    text-align: center;
    color: inherit;
  }

  .count-pill {
    border-radius: 999px;
    background: rgba(255, 255, 255, 0.12);
    padding: 1px 6px;
    font-size: 11px;
  }

  .search {
    border: 1px solid #2a3348;
    border-radius: 1.5rem;
    background: #0f1521;
    display: flex;
    align-items: center;
    padding: 0 12px;
    min-width: 250px;
    max-width: 360px;
    width: 100%;
    flex: 0 1 360px;
    gap: 8px;
  }

  .search-icon {
    color: #8f9aaa;
    font-size: 13px;
  }

  .search input {
    border: none;
    background: transparent;
    color: #dce2ed;
    width: 100%;
    height: 36px;
    outline: none;
    font-size: 13px;
  }

  .search input::placeholder {
    color: #7f899b;
  }

  .hidden-file-input {
    display: none;
  }

  .notice {
    margin: 8px 14px 0;
    padding: 8px 10px;
    border-radius: 8px;
    font-size: 12px;
  }

  .notice.error {
    color: #ffccd4;
    background: rgba(165, 42, 42, 0.24);
    border: 1px solid rgba(255, 99, 132, 0.35);
  }

  .notice.ok {
    color: #d8ffe8;
    background: rgba(28, 120, 72, 0.24);
    border: 1px solid rgba(71, 211, 125, 0.45);
  }

  .table-shell {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    margin: 10px 12px;
    border: 1px solid #232d42;
    border-radius: 10px;
    overflow: hidden;
    background: #101621;
  }

  .grid-row {
    display: grid;
    grid-template-columns:
      34px
      minmax(220px, 2.3fr)
      minmax(95px, 0.9fr)
      minmax(220px, 1.8fr)
      minmax(118px, 1fr)
      minmax(95px, 0.95fr)
      minmax(95px, 0.95fr)
      minmax(110px, 1fr)
      minmax(110px, 1fr)
      minmax(72px, 0.7fr)
      34px;
    align-items: center;
    gap: 10px;
    min-width: 1080px;
  }

  .table-header {
    background: #0b111a;
    border-bottom: 1px solid #24304a;
    padding: 10px 12px;
    text-transform: uppercase;
    font-size: 11px;
    letter-spacing: 0.08em;
    color: #94a0b5;
    font-weight: 600;
  }

  .th {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .torrent-select-toggle {
    appearance: none;
    width: 18px;
    height: 18px;
    border-radius: 999px;
    border: 4px solid #3f4655;
    background: #6d7483;
    cursor: pointer;
    margin: 0;
    transition:
      background-color 120ms ease,
      border-color 120ms ease,
      box-shadow 120ms ease;
  }

  .torrent-select-toggle:checked {
    background: #ffae00;
    border-color: #8e5f00;
  }

  .torrent-select-toggle:focus-visible {
    outline: none;
    box-shadow: 0 0 0 2px rgba(255, 174, 0, 0.3);
  }

  .sortable .caret {
    opacity: 0.75;
  }

  .sortable.active {
    color: #ffae00;
  }

  .sortable.active .caret {
    color: #ffae00;
  }

  .speed-header.down .speed-arrow {
    color: #5fe28f;
  }

  .speed-header.up .speed-arrow {
    color: #f7a95c;
  }

  .table-body {
    overflow: auto;
    flex: 1;
  }

  .table-row {
    padding: 10px 12px;
    border-bottom: 1px solid #1e283d;
    font-size: 13px;
    color: #dfe5f0;
  }

  .table-row:hover {
    background: #171f2f;
  }

  .cell {
    min-width: 0;
  }

  .name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .numeric,
  .size {
    color: #c6cedb;
    font-variant-numeric: tabular-nums;
  }

  .progress-cell {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .progress-complete {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    color: #b9f5cf;
    font-weight: 600;
  }

  .complete-icon {
    color: #47d37d;
    font-size: 11px;
  }

  .progress-active {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .bar-track {
    width: 100%;
    height: 6px;
    border-radius: 999px;
    background: #27324a;
    overflow: hidden;
  }

  .bar-fill {
    height: 100%;
    border-radius: 999px;
    background: linear-gradient(90deg, #2fb56b, #5fe28f);
  }

  .progress-meta {
    display: flex;
    justify-content: space-between;
    color: #8c97aa;
    font-size: 11px;
    font-variant-numeric: tabular-nums;
  }

  .status-pill {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 999px;
    padding: 4px 10px;
    font-size: 11px;
    font-weight: 700;
  }

  .status-pill.downloading {
    color: #d7ffe7;
    background: #1f6a45;
  }

  .status-pill.seeding {
    color: #e7ffef;
    background: #2b8a58;
  }

  .status-pill.stopped {
    color: #aeb7c7;
    background: #394456;
  }

  .status-pill.paused {
    color: #d7dded;
    background: #4a5872;
  }

  .speed-down {
    color: #9af0bf;
  }

  .speed-up {
    color: #ffc58b;
  }

  .actions-col {
    display: flex;
    justify-content: center;
  }

  .row-action {
    border: none;
    background: transparent;
    color: #a2afc2;
    font-size: 20px;
    line-height: 1;
    padding: 0;
    width: 20px;
    height: 20px;
  }

  .empty-state {
    padding: 24px;
    text-align: center;
    color: #909bb0;
    font-size: 13px;
  }

  .status-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 10px 14px;
    border-top: 1px solid #222a3b;
    background: #10151f;
    font-size: 12px;
    color: #b3bdcd;
    gap: 10px;
  }

  .status-left {
    display: inline-flex;
    align-items: center;
    gap: 18px;
  }

  .status-right {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    flex-wrap: wrap;
    justify-content: flex-end;
  }

  .selection-actions {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  .action,
  .hidden-btn {
    border: 1px solid #2f3951;
    background: #1a2232;
    color: #e2e8f3;
    border-radius: 999px;
    padding: 6px 12px;
    font-size: 12px;
    font-weight: 600;
  }

  .action.pause {
    border-color: #3a4a67;
  }

  .action.archive {
    border-color: #6b5a3a;
    color: #f3ddba;
  }

  .action.delete {
    border-color: #6f3f4a;
    color: #ffc8d0;
  }

  .action.hide,
  .hidden-btn {
    border-color: #6b5a3a;
  }

  .hidden-count {
    margin-left: 6px;
    background: rgba(255, 174, 0, 0.2);
    color: #ffd288;
    padding: 1px 6px;
    border-radius: 999px;
  }

  .connected {
    display: inline-flex;
    align-items: center;
    gap: 7px;
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #47d37d;
    box-shadow: 0 0 0 4px rgba(71, 211, 125, 0.2);
  }

  .agg {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-variant-numeric: tabular-nums;
  }

  .agg.down {
    color: #76e6a7;
  }

  .agg.up {
    color: #ffc58b;
  }

  .add-modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: 10000;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
    background: rgba(5, 9, 14, 0.72);
    backdrop-filter: blur(7px);
  }

  .add-modal {
    width: min(700px, 100%);
    border-radius: 16px;
    border: 1px solid #2a3448;
    background: #131b28;
    box-shadow: 0 24px 58px rgba(0, 0, 0, 0.45);
    overflow: hidden;
  }

  .add-modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 16px 12px;
  }

  .add-modal-title-wrap {
    display: inline-flex;
    align-items: center;
    gap: 10px;
  }

  .add-modal-badge {
    width: 30px;
    height: 30px;
    border-radius: 8px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: #302413;
    border: 1px solid #a67b3f;
    color: #ffae00;
    font-size: 18px;
    font-weight: 700;
    line-height: 1;
  }

  .add-modal-title-wrap h2 {
    margin: 0;
    font-size: 20px;
    font-weight: 700;
  }

  .add-modal-close {
    width: 30px;
    height: 30px;
    border: 1px solid #33405a;
    border-radius: 8px;
    background: #1a2436;
    color: #c7d0de;
    font-size: 14px;
    line-height: 1;
  }

  .add-modal-divider {
    height: 1px;
    background: #26324a;
  }

  .add-modal-body {
    padding: 14px 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .add-mode-track {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 4px;
    border-radius: 999px;
    border: 1px solid #2f3b55;
    background: #0f1624;
    padding: 4px;
  }

  .add-mode-segment {
    border: none;
    border-radius: 999px;
    background: transparent;
    color: #a9b4c7;
    padding: 9px 12px;
    font-size: 13px;
    font-weight: 600;
  }

  .add-mode-segment.active {
    background: #4f402a;
    color: #f6e6cd;
  }

  .add-field-label {
    color: #aeb8cb;
    font-size: 12px;
    font-weight: 600;
  }

  .add-dropzone {
    width: 100%;
    min-height: 152px;
    border-radius: 12px;
    border: 1px dashed #8f7348;
    background: #151f31;
    color: #dbe4f2;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 16px;
  }

  .add-dropzone.drag-active {
    border-color: #ffae00;
    background: #1c2840;
  }

  .add-drop-icon {
    font-size: 26px;
    color: #ffae00;
    line-height: 1;
  }

  .add-drop-title {
    font-size: 15px;
    font-weight: 600;
  }

  .add-drop-subtitle {
    font-size: 12px;
    color: #95a2b7;
  }

  .add-selected-files {
    margin: 0;
    font-size: 12px;
    color: #b8c2d5;
  }

  .add-input,
  .add-meta-field select {
    width: 100%;
    border: 1px solid #34415c;
    border-radius: 10px;
    background: #101827;
    color: #e1e8f4;
    padding: 10px 12px;
    font-size: 13px;
  }

  .add-help-text {
    margin: -2px 0 0;
    font-size: 12px;
    color: #9aa6bb;
  }

  .add-meta-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .add-meta-field {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .add-meta-field > span {
    font-size: 12px;
    color: #aeb8cb;
    font-weight: 600;
  }

  .add-options-row {
    display: flex;
    align-items: center;
    gap: 18px;
    flex-wrap: wrap;
  }

  .add-checkbox {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    color: #c4cedf;
    font-size: 13px;
  }

  .add-checkbox input {
    width: 14px;
    height: 14px;
  }

  .add-error {
    margin: 0;
    padding: 9px 10px;
    border-radius: 8px;
    color: #ffccd4;
    background: rgba(165, 42, 42, 0.24);
    border: 1px solid rgba(255, 99, 132, 0.35);
    font-size: 12px;
  }

  .add-modal-footer {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
    padding: 0 16px 16px;
  }

  .add-footer-btn {
    border-radius: 10px;
    border: 1px solid #33405a;
    padding: 10px 12px;
    font-size: 13px;
    font-weight: 700;
  }

  .add-footer-btn.secondary {
    background: #1a2436;
    color: #d2dcea;
  }

  .add-footer-btn.primary {
    background: #ffae00;
    border-color: #ffae00;
    color: #2b1f0f;
  }

  .add-footer-btn:disabled,
  .add-modal-close:disabled,
  .add-mode-segment:disabled,
  .add-dropzone:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: 10001;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.62);
  }

  .modal-panel {
    width: min(480px, 94vw);
    border-radius: 12px;
    border: 1px solid #2b344a;
    background: #121925;
    padding: 14px;
  }

  .modal-panel h3 {
    margin: 0 0 8px;
    font-size: 18px;
  }

  .modal-panel p {
    margin: 0 0 10px;
    color: #afbacd;
    font-size: 13px;
  }

  .modal-panel input {
    width: 100%;
    border: 1px solid #2e3850;
    background: #0f1521;
    color: #e8edf8;
    border-radius: 8px;
    padding: 10px;
    margin-bottom: 12px;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .modal-actions button {
    border: 1px solid #2f3951;
    background: #1a2232;
    color: #dbe4f3;
    border-radius: 8px;
    padding: 7px 12px;
    font-size: 12px;
  }

  .modal-actions .confirm {
    border-color: #5f8a6d;
    background: #1f6a45;
    color: #ddffeb;
  }

  @media (max-width: 1100px) {
    .controls-row {
      flex-direction: column;
      align-items: stretch;
    }

    .search {
      max-width: none;
      flex-basis: auto;
    }
  }

  @media (max-width: 720px) {
    .control-cluster {
      flex-wrap: wrap;
      overflow-x: visible;
    }

    .status-bar {
      flex-flow: row wrap;
    }

    .status-right {
      width: 100%;
      justify-content: flex-start;
    }

    .add-modal {
      width: 100%;
    }

    .add-meta-grid,
    .add-modal-footer {
      grid-template-columns: 1fr;
    }
  }
</style>
