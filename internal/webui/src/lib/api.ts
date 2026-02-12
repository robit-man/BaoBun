import type {
  HiddenCountResponse,
  SeedConfig,
  BaoActionKind,
  BaoActionResponse,
  BaoStatus,
  UploadBaoResponse,
} from "./types";

export async function fetchBaos(): Promise<BaoStatus[]> {
  const res = await fetch("/api/v1/baos");
  if (!res.ok) {
    throw new Error("failed to fetch baos");
  }

  const data = await res.json();
  if (!Array.isArray(data)) {
    return [];
  }

  return data.map((bao) => ({
    ...bao,
    downloaded: Number(bao?.downloaded ?? 0),
    uploaded: Number(bao?.uploaded ?? 0),
    ratio: Number(bao?.ratio ?? 0),
    peers: Array.isArray(bao?.peers) ? bao.peers : [],
    files: Array.isArray(bao?.files) ? bao.files : [],
  })) as BaoStatus[];
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

export async function applyBaoAction(
  action: BaoActionKind,
  ids: string[],
  passkey?: string
): Promise<BaoActionResponse> {
  const body: Record<string, unknown> = { ids };
  if (passkey) {
    body.passkey = passkey;
  }

  const res = await fetch(`/api/v1/baos/actions/${action}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `failed to ${action} baos`);
  }

  return res.json();
}

export async function unhideBaos(
  passkey: string
): Promise<BaoActionResponse> {
  const res = await fetch("/api/v1/baos/hidden/unhide", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ passkey }),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || "failed to unhide baos");
  }

  return res.json();
}

export async function fetchHiddenCount(): Promise<number> {
  const res = await fetch("/api/v1/baos/hidden/count");
  if (!res.ok) {
    throw new Error("failed to fetch hidden count");
  }

  const payload = (await res.json()) as HiddenCountResponse;
  return Number(payload?.count ?? 0);
}
