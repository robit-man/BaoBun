import type { SeedConfig, TorrentStatus } from "./types";

export async function fetchTorrents(): Promise<TorrentStatus[]> {
  const res = await fetch("/api/v1/torrents");
  if (!res.ok) {
    throw new Error("failed to fetch torrents");
  }
  return res.json();
}

export async function fetchSeedConfig(): Promise<SeedConfig> {
  const res = await fetch("/api/v1/config/seeds");
  if (!res.ok) {
    throw new Error("failed to fetch seed config");
  }
  return res.json();
}

export async function saveSeedConfig(seeds: string[]): Promise<SeedConfig> {
  const res = await fetch("/api/v1/config/seeds", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ seeds }),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || "failed to save seed config");
  }

  return res.json();
}

export async function autoGenerateSeedConfig(): Promise<SeedConfig> {
  const res = await fetch("/api/v1/config/seeds/generate", {
    method: "POST",
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || "failed to auto-generate seed config");
  }

  return res.json();
}
