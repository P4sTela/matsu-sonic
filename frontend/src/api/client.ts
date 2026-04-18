const BASE = "/api";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || `HTTP ${res.status}`);
  }
  return res.json();
}

// Config
export const getConfig = () => request<import("./types").Config>("/config");
export const updateConfig = (data: Partial<import("./types").Config>) =>
  request<{ status: string }>("/config", {
    method: "POST",
    body: JSON.stringify(data),
  });

// Auth
export const testAuth = () =>
  request<{ status: string; user: { displayName: string; emailAddress: string } }>("/auth/test", { method: "POST" });

// Sync
export const startFullSync = () =>
  request<{ status: string; mode: string }>("/sync/full", { method: "POST" });
export const startIncrementalSync = () =>
  request<{ status: string; mode: string }>("/sync/incremental", { method: "POST" });
export const cancelSync = () =>
  request<{ status: string }>("/sync/cancel", { method: "POST" });
export const resetSync = () =>
  request<{ status: string }>("/sync/reset", { method: "POST" });
export const getSyncDiff = () =>
  request<import("./types").DiffEntry[]>("/sync/diff");
export const getSyncStatus = () =>
  request<{ is_running: boolean; progress: import("./types").SyncProgress }>("/sync/status");
export const getSyncHistory = (limit = 20) =>
  request<import("./types").SyncRun[]>(`/sync/history?limit=${limit}`);

// Files
export const listFiles = (search = "") =>
  request<import("./types").SyncedFile[]>(`/files${search ? `?search=${encodeURIComponent(search)}` : ""}`);
export const deleteFiles = (fileIds: string[]) =>
  request<{ deleted: number }>("/files/delete", {
    method: "POST",
    body: JSON.stringify({ file_ids: fileIds }),
  });
export const getFile = (id: string) =>
  request<import("./types").SyncedFile>(`/files/${id}`);
export const verifyFiles = () =>
  request<import("./types").VerifyResponse>("/files/verify", { method: "POST" });
export const resyncFiles = (fileIds: string[]) =>
  request<{ status: string; cleared: number }>("/files/resync", {
    method: "POST",
    body: JSON.stringify({ file_ids: fileIds }),
  });

// Revisions
export const listRevisions = (fileId: string) =>
  request<import("./types").DriveRevision[]>(`/files/${fileId}/revisions`);
export const downloadRevision = (fileId: string, revId: string, destDir?: string) =>
  request<{ path: string; size: number }>(`/files/${fileId}/revisions/${revId}/download`, {
    method: "POST",
    body: JSON.stringify({ dest_dir: destDir }),
  });
export const listDownloadedRevisions = (fileId: string) =>
  request<import("./types").DownloadedRevision[]>(`/files/${fileId}/revisions/downloaded`);

// Distribution
export const listTargets = () =>
  request<import("./types").DistTarget[]>("/distribution/targets");
export const addTarget = (target: import("./types").DistTarget) =>
  request<{ status: string }>("/distribution/targets", {
    method: "POST",
    body: JSON.stringify(target),
  });
export const removeTarget = (name: string) =>
  request<{ status: string }>(`/distribution/targets/${name}`, { method: "DELETE" });
export const testTarget = (name: string) =>
  request<{ status: string }>(`/distribution/targets/${name}/test`, { method: "POST" });
export const distribute = (fileIds: string[], targetName: string, destDir?: string) =>
  request<{ status: string }>("/distribute", {
    method: "POST",
    body: JSON.stringify({ file_ids: fileIds, target_name: targetName, dest_dir: destDir }),
  });
export const listDistJobs = (limit = 50) =>
  request<import("./types").DistJob[]>(`/distribution/jobs?limit=${limit}`);

// Browse
export const browseDirectory = (path?: string) =>
  request<import("./types").BrowseResult>(`/browse${path ? `?path=${encodeURIComponent(path)}` : ""}`);
export const makeDirectory = (path: string) =>
  request<{ status: string; path: string }>("/mkdir", {
    method: "POST",
    body: JSON.stringify({ path }),
  });

// Drive Browse
export const browseDrive = (folderId?: string, source?: "my_drive" | "shared") => {
  const params = new URLSearchParams();
  if (folderId) params.set("folder_id", folderId);
  if (source) params.set("source", source);
  const qs = params.toString();
  return request<import("./types").DriveBrowseResult>(`/drive/browse${qs ? `?${qs}` : ""}`);
};
