import type {
  HiddenCountResponse,
  SeedConfig,
  TorrentActionKind,
  TorrentActionResponse,
  TorrentStatus,
  UploadBaoResponse,
} from "./types";

export async function fetchTorrents(): Promise<TorrentStatus[]> {
  const res = await fetch("/api/v1/torrents");
  if (!res.ok) {
    throw new Error("failed to fetch torrents");
  }

  const data = await res.json();
  if (!Array.isArray(data)) {
    return [];
  }

  return data.map((torrent) => ({
    ...torrent,
    downloaded: Number(torrent?.downloaded ?? 0),
    uploaded: Number(torrent?.uploaded ?? 0),
    ratio: Number(torrent?.ratio ?? 0),
    peers: Array.isArray(torrent?.peers) ? torrent.peers : [],
    files: Array.isArray(torrent?.files) ? torrent.files : [],
  })) as TorrentStatus[];
}

export async function uploadBao(
  fileName: string,
  data: Uint8Array
): Promise<UploadBaoResponse> {
  const res = await fetch("/api/v1/bao", {
    method: "POST",
    headers: {
      "Content-Type": "application/octet-stream",
      "X-Filename": encodeURIComponent(fileName),
    },
    body: data,
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || "failed to upload bao");
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

export async function applyTorrentAction(
  action: TorrentActionKind,
  ids: string[],
  passkey?: string
): Promise<TorrentActionResponse> {
  const body: Record<string, unknown> = { ids };
  if (passkey) {
    body.passkey = passkey;
  }

  const res = await fetch(`/api/v1/torrents/actions/${action}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `failed to ${action} torrents`);
  }

  return res.json();
}

export async function unhideTorrents(
  passkey: string
): Promise<TorrentActionResponse> {
  const res = await fetch("/api/v1/torrents/hidden/unhide", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ passkey }),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || "failed to unhide torrents");
  }

  return res.json();
}

export async function fetchHiddenCount(): Promise<number> {
  const res = await fetch("/api/v1/torrents/hidden/count");
  if (!res.ok) {
    throw new Error("failed to fetch hidden count");
  }

  const payload = (await res.json()) as HiddenCountResponse;
  return Number(payload?.count ?? 0);
}
