<script lang="ts">
  import { createEventDispatcher } from "svelte";

  export let open = false;
  export let seeds: string[] = [];
  export let seedLength = 32;
  export let seedCount = 4;
  export let busy = false;
  export let error: string | null = null;
  export let message: string | null = null;

  const dispatch = createEventDispatcher();

  let localError: string | null = null;
  let draftSeeds: string[] = [];

  $: if (open) {
    draftSeeds = Array.from({ length: seedCount }, (_, i) => seeds[i] ?? "");
    localError = null;
  }

  function close() {
    dispatch("close");
  }

  function onBackdropKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      close();
    }
  }

  function validate() {
    if (draftSeeds.length !== seedCount) {
      return `Expected ${seedCount} seeds`;
    }

    for (let i = 0; i < draftSeeds.length; i += 1) {
      const seed = draftSeeds[i] ?? "";
      if (seed.length !== seedLength) {
        return `Seed ${i + 1} must be exactly ${seedLength} characters`;
      }
    }

    return null;
  }

  function save() {
    localError = validate();
    if (localError) {
      return;
    }
    dispatch("save", { seeds: [...draftSeeds] });
  }

  function generate() {
    localError = null;
    dispatch("generate");
  }
</script>

{#if open}
  <div
    class="backdrop"
    role="button"
    tabindex="0"
    on:click|self={close}
    on:keydown={onBackdropKeydown}
  >
    <div class="modal">
      <h2>Seed Configuration</h2>
      <p class="meta">
        Configure {seedCount} NKN seeds. Each seed must be exactly {seedLength}
        characters.
      </p>

      <div class="grid">
        {#each draftSeeds as _, i}
          <label class="field">
            <span>Seed {i + 1}</span>
            <input
              type="text"
              bind:value={draftSeeds[i]}
              maxlength={seedLength}
              spellcheck="false"
              autocomplete="off"
              disabled={busy}
            />
          </label>
        {/each}
      </div>

      {#if localError}
        <p class="status error">{localError}</p>
      {/if}
      {#if error}
        <p class="status error">{error}</p>
      {/if}
      {#if message}
        <p class="status ok">{message}</p>
      {/if}

      <p class="meta">Changes apply after restarting the client.</p>

      <div class="actions">
        <button type="button" on:click={close} disabled={busy}>Close</button>
        <button type="button" on:click={generate} disabled={busy}>
          Auto Generate + Save
        </button>
        <button type="button" on:click={save} disabled={busy}>Save Seeds</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
  }

  .modal {
    width: min(760px, 95vw);
    max-height: 90vh;
    overflow: auto;
    background: #12151f;
    border: 1px solid #2a3043;
    border-radius: 12px;
    padding: 16px;
    text-align: left;
  }

  h2 {
    margin: 0 0 8px;
    font-size: 20px;
  }

  .meta {
    margin: 6px 0 12px;
    color: var(--muted);
    font-size: 13px;
  }

  .grid {
    display: grid;
    gap: 10px;
  }

  .field {
    display: grid;
    gap: 6px;
    font-size: 12px;
  }

  .field span {
    color: var(--muted);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .field input {
    width: 100%;
    background: #0c0f18;
    border: 1px solid #2a3043;
    color: var(--text);
    border-radius: 8px;
    padding: 10px;
    font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
    font-size: 12px;
  }

  .status {
    margin: 10px 0 0;
    font-size: 13px;
  }

  .status.error {
    color: var(--red);
  }

  .status.ok {
    color: #47d37d;
  }

  .actions {
    margin-top: 14px;
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }
</style>
