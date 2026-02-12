<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import type { TorrentStatus } from "./types";
  import baoSwarmIcon from "../assets/baoswarm.png";

  type UploadFile = { name: string; data: Uint8Array };
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
  }>();

  let activeFilter: FilterId = "all";
  let searchTerm = "";
  let selectedIds = new Set<string>();
  let selectAllEl: HTMLInputElement | null = null;
  let filePickerEl: HTMLInputElement | null = null;

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

  $: if (selectAllEl) {
    selectAllEl.indeterminate = !allVisibleSelected && someVisibleSelected;
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

  function statusLabel(torrent: TorrentStatus): "Downloading" | "Seeding" | "Stopped" {
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

  async function triggerAddPicker() {
    if (filePickerEl) {
      filePickerEl.value = "";
      filePickerEl.click();
    }
  }

  async function onPickerChange(event: Event) {
    const input = event.currentTarget as HTMLInputElement | null;
    const files = Array.from(input?.files ?? []);
    if (files.length === 0) {
      return;
    }

    try {
      const payload: UploadFile[] = [];
      for (const file of files) {
        payload.push({
          name: file.name,
          data: new Uint8Array(await file.arrayBuffer()),
        });
      }
      dispatch("load", { files: payload });
    } finally {
      if (input) {
        input.value = "";
      }
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
      <button class="add-btn" type="button" on:click={triggerAddPicker}>
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

  <div class="table-shell">
    <div class="table-header grid-row">
      <div class="th select-col">
        <input
          bind:this={selectAllEl}
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
  </div>
</div>

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
    border-radius: 999px;
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
    border-radius: 999px;
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
    justify-content: flex-start;
    padding: 10px 14px;
    border-top: 1px solid #222a3b;
    background: #10151f;
    font-size: 12px;
    color: #b3bdcd;
  }

  .status-left {
    display: inline-flex;
    align-items: center;
    gap: 18px;
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
</style>
