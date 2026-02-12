<script lang="ts">
  import type { TorrentStatus } from "./types";

  export let torrents: TorrentStatus[] = [];
  export let selected: TorrentStatus | null = null;

  function select(t: TorrentStatus) {
    selected = t;
  }

  function onRowKeydown(event: KeyboardEvent, t: TorrentStatus) {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      select(t);
    }
  }

  function percent(t: TorrentStatus) {
    return Math.round(((t.fileSize - t.remaining) / t.fileSize) * 100);
  }

  function rate(n: number) {
    if (n < 1024) return `${n} B/s`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB/s`;
    return `${(n / 1024 / 1024).toFixed(1)} MB/s`;
  }

  function size(n: number) {
    if (n < 1024) return `${n} B`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(2)} KB`;
    return `${(n / 1024 / 1024).toFixed(2)} MB`;
  }

  function stateClass(state: string) {
    return `state ${state}`;
  }
</script>

<div class="torrent-list scroll">
  <div class="header">
    <span>Name</span>
    <span>Progress</span>
    <span>↓</span>
    <span>↑</span>
    <span>Peers</span>
    <span>Size</span>
    <span>Status</span>
  </div>

  <div class="rows">
    {#each torrents as t (t.id)}
      <div
        class="row {selected?.id === t.id ? 'selected' : ''}"
        role="button"
        tabindex="0"
        on:click={() => select(t)}
        on:keydown={(event) => onRowKeydown(event, t)}
      >
        <div class="name">{t.name}</div>
        <div class="progress">
          <div class="bar">
            <div class="fill" style="width: {percent(t)}%"></div>
          </div>
          <span class="pct">{percent(t)}%</span>
        </div>
        <div class="rate down">{rate(t.upRate)}</div>
        <div class="rate up">{rate(t.downRate)}</div>
        <div class="peers">{t.peers.length}</div>
        <div class="size">{size(t.fileSize)}</div>
        <div class={stateClass(t.state)}>{t.state}</div>
      </div>
    {/each}
  </div>
</div>

<style>
  .scroll {
    height: 100%;
    overflow: auto;
  }

  .rows {
    min-width: 900px; /* enables horizontal scroll */
  }

  .row.selected {
    background: #1b2340;
    outline: 1px solid var(--accent);
  }
</style>
