export type BaoState =
  | "downloading"
  | "seeding"
  | "stopped"
  | "paused"
  | "queued"
  | "stalled"
  | "error";

export interface BaoStatus {
  id: string;
  name: string;
  downRate: number;   // bytes/sec
  upRate: number;     // bytes/sec
  downloaded: number; // bytes
  uploaded: number;   // bytes
  ratio: number;
  peers: PeerStatus[];
  files: FileStatus[];
  state: BaoState;
  fileSize: number;
  remaining: number;
}

export interface FileStatus {
  path: string;
  length: number;
  remaining: number;
}

export interface PeerStatus {
  id: string;
  state: string;
  downRate: number;   // bytes/sec
  upRate: number;     // bytes/sec
}

export interface SeedConfig {
  seeds: string[];
  seedLength: number;
  seedCount: number;
  restartRequired: boolean;
}

export interface UploadBaoResponse {
  infoHash: string;
  name: string;
}

export type BaoActionKind = "pause" | "archive" | "delete" | "hide";

export interface BaoActionResponse {
  processed: number;
  hidden: number;
  remaining: number;
  successful: boolean;
  message: string;
}

export interface HiddenCountResponse {
  count: number;
}
