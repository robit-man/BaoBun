<script lang="ts">
  import { onMount } from "svelte";
  import TorrentList from "./lib/TorrentsList.svelte";
  import TorrentDetails from "./lib/TorrentDetails.svelte";
  import {
    autoGenerateSeedConfig,
    fetchSeedConfig,
    fetchTorrents,
    saveSeedConfig,
  } from "./lib/api";
  import type { SeedConfig, TorrentStatus } from "./lib/types";
  import FileDrop from "./components/FileDrop.svelte";
  import SeedConfigModal from "./components/SeedConfigModal.svelte";

  let torrents: TorrentStatus[] = [];
  let selected: TorrentStatus | null = null;
  let error: string | null = null;
  let configOpen = false;
  let configBusy = false;
  let configError: string | null = null;
  let configMessage: string | null = null;
  let seedConfig: SeedConfig | null = null;
  const REFRESH_INTERVAL_MS = 1000;

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
    loadSeedConfig();
    const id = setInterval(refresh, REFRESH_INTERVAL_MS);
    return () => clearInterval(id);
  });

  async function handleLoad(event: CustomEvent<{ data: Uint8Array }>) {
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

  async function openConfig() {
    configOpen = true;
    configError = null;
    configMessage = null;
    await loadSeedConfig();
  }

  async function loadSeedConfig() {
    try {
      seedConfig = await fetchSeedConfig();
    } catch {
      // Keep UI running if config endpoint is temporarily unavailable.
    }
  }

  async function onSaveSeeds(event: CustomEvent<{ seeds: string[] }>) {
    configBusy = true;
    configError = null;
    configMessage = null;

    try {
      seedConfig = await saveSeedConfig(event.detail.seeds);
      configMessage = "Saved. Restart the client to apply these seeds.";
    } catch (err) {
      configError = err instanceof Error ? err.message : "Failed to save seeds";
    } finally {
      configBusy = false;
    }
  }

  async function onGenerateSeeds() {
    configBusy = true;
    configError = null;
    configMessage = null;

    try {
      seedConfig = await autoGenerateSeedConfig();
      configMessage = "Generated and saved. Restart the client to apply.";
    } catch (err) {
      configError =
        err instanceof Error ? err.message : "Failed to auto-generate seeds";
    } finally {
      configBusy = false;
    }
  }
</script>

<FileDrop on:load={handleLoad} />

<SeedConfigModal
  open={configOpen}
  busy={configBusy}
  seeds={seedConfig?.seeds ?? []}
  seedLength={seedConfig?.seedLength ?? 32}
  seedCount={seedConfig?.seedCount ?? 4}
  error={configError}
  message={configMessage}
  on:close={() => (configOpen = false)}
  on:save={onSaveSeeds}
  on:generate={onGenerateSeeds}
/>

<div class="app-layout">
  <div class="toolbar">
    <div class="toolbar-left">
      <strong>BaoBun</strong>
    </div>
    <button class="toolbar-btn" type="button" on:click={openConfig}>
      Config
    </button>
  </div>

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
    padding: 8px;
    box-sizing: border-box;
  }

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: var(--panel);
    border-radius: 8px;
    padding: 8px 12px;
  }

  .toolbar-left {
    font-size: 14px;
  }

  .toolbar-btn {
    padding: 6px 10px;
    font-size: 12px;
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
