export type TorrentState =
  | "downloading"
  | "seeding"
  | "stopped"
  | "paused"
  | "queued"
  | "stalled"
  | "error";

export interface TorrentStatus {
  id: string;
  name: string;
  downRate: number;   // bytes/sec
  upRate: number;     // bytes/sec
  downloaded: number; // bytes
  uploaded: number;   // bytes
  ratio: number;
  peers: PeerStatus[];
  files: FileStatus[];
  state: TorrentState;
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

export type TorrentActionKind = "pause" | "archive" | "delete" | "hide";

export interface TorrentActionResponse {
  processed: number;
  hidden: number;
  remaining: number;
  successful: boolean;
  message: string;
}

export interface HiddenCountResponse {
  count: number;
}
