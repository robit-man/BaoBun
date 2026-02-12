<script lang="ts">
  import type { BaoStatus } from "./types";

  export let bao: BaoStatus | null = null;
  let tab: "peers" | "files" = "peers";

  function rate(n: number) {
    if (n < 1024) return `${n} B/s`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB/s`;
    return `${(n / 1024 / 1024).toFixed(1)} MB/s`;
  }

  function size(n: number) {
    if (n < 1024) return `${n} B`;
    if (n < 1024 * 1024) return `${(n / 1024).toFixed(2)} KB`;
    if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(2)} MB`;
    return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
  }

  function fileProgress(length: number, remaining: number) {
    if (length === 0) return 0;
    return Math.round(((length - remaining) / length) * 100);
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
    {#if !bao}
      <div class="empty">Select a bao to see details</div>
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
          {#each [...(bao.peers ?? [])].sort((a, b) => a.id.localeCompare(b.id)) as p}
            <tr>
              <td>{p.id}</td>
              <td>{p.state}</td>
              <td>{rate(p.downRate)}</td>
              <td>{rate(p.upRate)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {:else if !bao.files?.length}
      <div class="empty">No files found for this bao</div>
    {:else}
      <table>
        <thead>
          <tr>
            <th>Path</th>
            <th>Size</th>
            <th>Remaining</th>
            <th>Progress</th>
          </tr>
        </thead>
        <tbody>
          {#each bao.files ?? [] as f}
            <tr>
              <td>{f.path}</td>
              <td>{size(f.length)}</td>
              <td>{size(f.remaining)}</td>
              <td>{fileProgress(f.length, f.remaining)}%</td>
            </tr>
          {/each}
        </tbody>
      </table>
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
