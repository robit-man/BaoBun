<script lang="ts">
  import { onMount } from "svelte";
  import {
    autoGenerateSeedConfig,
    fetchSeedConfig,
    fetchBaos,
    saveSeedConfig,
    uploadBao,
  } from "./lib/api";
  import type { SeedConfig, BaoStatus } from "./lib/types";
  import FileDrop from "./components/FileDrop.svelte";
  import SeedConfigModal from "./components/SeedConfigModal.svelte";
  import BaosList from "./lib/BaosList.svelte";

  let baos: BaoStatus[] = [];
  let error: string | null = null;
  let configOpen = false;
  let configBusy = false;
  let configError: string | null = null;
  let configMessage: string | null = null;
  let seedConfig: SeedConfig | null = null;
  let uploadError: string | null = null;
  let uploadMessage: string | null = null;

  const REFRESH_INTERVAL_MS = 1000;

  async function refresh() {
    try {
      baos = await fetchBaos();
      error = null;
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

  async function handleLoad(
    event: CustomEvent<{ files: { name: string; data: Uint8Array }[] }>
  ) {
    uploadError = null;
    uploadMessage = null;

    const files = event.detail.files ?? [];
    if (files.length === 0) {
      return;
    }

    try {
      for (const file of files) {
        await uploadBao(file.name, file.data);
      }

      await refresh();
      uploadMessage = `Imported ${files.length} file(s).`;
    } catch (err) {
      uploadError =
        err instanceof Error ? err.message : "Failed to import dropped file(s)";
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

<div class="page-shell">
  <BaosList
    {baos}
    {error}
    {uploadError}
    {uploadMessage}
    on:load={handleLoad}
    on:refresh={refresh}
    on:openConfig={openConfig}
  />
</div>

<style>
  .page-shell {
    min-height: 100vh;
    padding: 14px;
    box-sizing: border-box;
  }
</style>
