<script lang="ts">
  import { onMount } from "svelte";
  import TorrentList from "./lib/TorrentsList.svelte";
  import TorrentDetails from "./lib/TorrentDetails.svelte";
  import { fetchTorrents } from "./lib/api";
  import type { TorrentStatus } from "./lib/types";
  import FileDrop from "./components/FileDrop.svelte";

  let torrents: TorrentStatus[] = [];
  let selected: TorrentStatus | null = null;
  let error: string | null = null;

  async function refresh() {
    try {
      torrents = await fetchTorrents();
      error = null;

      // keep selection in sync
      if (selected) {
        selected = torrents.find((t) => t.id === selected?.id) ?? selected;
      }
    } catch {
      error = "Disconnected";
    }
  }

  onMount(() => {
    refresh();
    const id = setInterval(refresh, 1000 / 60);
    return () => clearInterval(id);
  });

  async function handleLoad(event) {
    const res = await fetch("/api/v1/bao", {
      method: "POST",
      headers: {
        "Content-Type": "application/octet-stream",
      },
      body: event.detail.data.buffer,
    });
    //        "X-Filename": encodeURIComponent(file.name),

    if (!res.ok) {
      throw new Error("failed to upload file");
    }
  }
</script>

<FileDrop on:load={handleLoad} />

<div class="app-layout">
  {#if error}
    <p class="error">{error}</p>
  {/if}

  <div class="list-pane">
    <TorrentList {torrents} bind:selected />
  </div>

  <div class="details-pane">
    <TorrentDetails torrent={selected} />
  </div>
</div>

<style>
  .app-layout {
    height: 100vh;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .list-pane {
    flex: 1;
    min-height: 0; /* REQUIRED for scrolling */
  }

  .details-pane {
    height: 260px;
    background: var(--panel);
    border-radius: 8px;
    overflow: hidden;
  }

  .error {
    color: var(--red);
  }
</style>
