export interface Config {
  auth_method: "oauth" | "service_account";
  credentials_path: string;
  token_path: string;
  sync_folder_id: string;
  local_sync_dir: string;
  chunk_size_mb: number;
  max_workers: number;
  ignore_patterns: string[];
  distribution_targets: DistTarget[];
}

export interface DistTarget {
  name: string;
  type: "local" | "smb";
  path: string;
  server?: string;
  share?: string;
  username?: string;
}

export interface SyncedFile {
  file_id: string;
  name: string;
  mime_type: string;
  md5_checksum: string;
  size: number;
  drive_modified: string;
  local_path: string;
  last_synced: string;
  parent_id: string;
  is_folder: boolean;
}

export interface ActiveDownload {
  file_id: string;
  file_name: string;
  progress: number; // 0.0 ~ 1.0
}

export interface SyncProgress {
  total_files: number;
  completed_files: number;
  failed_files: number;
  skipped_files: number;
  bytes_downloaded: number;
  current_file: string;
  current_file_progress: number;
  active_downloads: ActiveDownload[];
  is_running: boolean;
  errors: string[];
}

export interface SyncRun {
  id: number;
  started_at: string;
  finished_at: string;
  status: "running" | "completed" | "failed" | "cancelled";
  files_synced: number;
  files_failed: number;
  bytes_downloaded: number;
}

export interface DriveRevision {
  id: string;
  modifiedTime: string;
  size: string;
  lastModifyingUser?: { displayName: string };
  mimeType: string;
  keepForever: boolean;
  originalFilename: string;
}

export interface DownloadedRevision {
  id: number;
  file_id: string;
  revision_id: string;
  local_path: string;
  downloaded_at: string;
  size: number;
}

export interface DistJob {
  id: number;
  file_id: string;
  source_path: string;
  target_type: string;
  target_path: string;
  status: "pending" | "completed" | "failed";
  created_at: string;
  error_message: string;
}

export interface BrowseResult {
  current: string;
  parent: string;
  cwd: string;
  items: { name: string; path: string; is_dir: boolean }[];
}

export interface DiffEntry {
  file_id: string;
  name: string;
  mime_type: string;
  size: number;
  drive_modified: string;
  local_path: string;
  action: "new" | "update" | "delete";
}

export interface DriveBrowseResult {
  folder_id: string;
  folder_name: string;
  parent_id: string;
  items: { id: string; name: string; is_folder: boolean; mime_type: string }[];
}

export interface SyncComplete {
  status: string;
  files_synced: number;
  files_failed: number;
  bytes_downloaded: number;
  duration_ms: number;
}

export interface LogEntry {
  level: "info" | "warn" | "error";
  msg: string;
  ts: string;
}

export type WSMessage =
  | { type: "sync_progress"; data: SyncProgress }
  | { type: "sync_complete"; data: SyncComplete }
  | { type: "log"; data: LogEntry }
  | { type: "pong" };
