import { useState, useCallback } from "react";
import type { SyncProgress, LogEntry, WSMessage } from "@/api/types";
import { useWebSocket } from "./useWebSocket";
import * as api from "@/api/client";

const emptyProgress: SyncProgress = {
  total_files: 0,
  completed_files: 0,
  failed_files: 0,
  skipped_files: 0,
  bytes_downloaded: 0,
  current_file: "",
  current_file_progress: 0,
  active_downloads: [],
  is_running: false,
  errors: [],
};

export function useSync() {
  const [progress, setProgress] = useState<SyncProgress>(emptyProgress);
  const [logs, setLogs] = useState<LogEntry[]>([]);

  const onMessage = useCallback((msg: WSMessage) => {
    switch (msg.type) {
      case "sync_progress":
        setProgress(msg.data);
        break;
      case "sync_complete":
        setProgress((prev) => ({ ...prev, is_running: false }));
        break;
      case "log":
        setLogs((prev) => [msg.data, ...prev].slice(0, 200));
        break;
    }
  }, []);

  useWebSocket(onMessage);

  const startFull = useCallback(async () => {
    await api.startFullSync();
    setProgress((prev) => ({ ...prev, is_running: true }));
  }, []);

  const startIncremental = useCallback(async () => {
    await api.startIncrementalSync();
    setProgress((prev) => ({ ...prev, is_running: true }));
  }, []);

  const cancel = useCallback(async () => {
    await api.cancelSync();
  }, []);

  return { progress, logs, startFull, startIncremental, cancel };
}

export type SyncState = ReturnType<typeof useSync>;
