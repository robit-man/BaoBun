<script lang="ts">
  import type { TorrentStatus } from "./types";

  export let torrent: TorrentStatus | null = null;
  let tab: "peers" | "files" = "peers";

  function rate(n: number) {
    if (n < 1024) return `${n} B/s`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB/s`;
    return `${(n / 1024 / 1024).toFixed(1)} MB/s`;
  }
</script>

<div class="details">
  <div class="tabs">
    <button class:active={tab === "peers"} on:click={() => (tab = "peers")}>
      Peers
    </button>
    <button class:active={tab === "files"} on:click={() => (tab = "files")}>
      Files
    </button>
  </div>

  <div class="content">
    {#if !torrent}
      <div class="empty">Select a torrent to see details</div>
    {:else if tab === "peers"}
      <table>
        <thead>
          <tr>
            <th>Address</th>
            <th>State</th>
            <th>↓</th>
            <th>↑</th>
          </tr>
        </thead>

        <tbody>
          {#each torrent.peers as p}
            <tr>
              <td>{p.id}</td>
              <td>{p.state}</td>
              <td>{rate(p.upRate)}</td>
              <td>{rate(p.downRate)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {:else}
      <div class="empty">File list not implemented yet</div>
    {/if}
  </div>
</div>

<style>
  .details {
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  .tabs {
    display: flex;
    border-bottom: 1px solid #1f2433;
    background: #0b0d12;
  }

  button {
    background: none;
    border: none;
    padding: 10px 14px;
    color: var(--muted);
    cursor: pointer;
    font-family: var(--font-ui);
  }

  button.active {
    color: var(--text);
    border-bottom: 2px solid var(--accent);
  }

  .content {
    flex: 1;
    overflow: auto;
    padding: 12px;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 12px;
  }

  thead th {
    text-align: left;
    color: var(--muted);
    font-weight: 500;
    border-bottom: 1px solid #1f2433;
  }

  th,
  td {
    padding: 6px 8px;
    border-bottom: 1px solid #1f2433;
  }

  tbody tr:hover {
    background: var(--row-hover);
  }

  .empty {
    color: var(--muted);
    padding: 16px;
  }
</style>
