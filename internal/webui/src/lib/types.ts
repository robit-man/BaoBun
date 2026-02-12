export type TorrentState =
  | "downloading"
  | "seeding"
  | "paused"
  | "queued"
  | "error";

export interface TorrentStatus {
  id: string;
  name: string;
  downRate: number;   // bytes/sec
  upRate: number;     // bytes/sec
  peers: PeerStatus[];
  state: TorrentState;
  fileSize: number;
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
