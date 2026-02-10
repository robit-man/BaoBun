import type { TorrentStatus } from "./types";

export async function fetchTorrents(): Promise<TorrentStatus[]> {
  const res = await fetch("/api/v1/torrents");
  if (!res.ok) {
    throw new Error("failed to fetch torrents");
  }
  return res.json();
}
