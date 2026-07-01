import { useState, useCallback } from "react";
import type {
	SyncProgress,
	VerifyProgress,
	WSMessage,
	Conflict,
} from "@/api/types";
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
	const [verifyProgress, setVerifyProgress] = useState<VerifyProgress | null>(
		null,
	);
	const [conflicts, setConflicts] = useState<Conflict[]>([]);
	const [skippedConflicts, setSkippedConflicts] = useState<Conflict[]>([]);

	const onMessage = useCallback((msg: WSMessage) => {
		switch (msg.type) {
			case "sync_progress":
				setProgress(msg.data);
				break;
			case "sync_complete":
				setProgress((prev) => ({ ...prev, is_running: false }));
				break;
			case "verify_progress":
				setVerifyProgress(msg.data);
				break;
			case "verify_complete":
				setVerifyProgress(null);
				break;
			case "conflicts_detected":
				setConflicts(msg.data);
				break;
			case "conflict_skipped":
				setSkippedConflicts((prev) => [...prev, msg.data]);
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

	const refreshConflicts = useCallback(async () => {
		const data = await api.getConflicts();
		setConflicts(data);
		return data;
	}, []);

	const clearSkippedConflicts = useCallback(() => {
		setSkippedConflicts([]);
	}, []);

	return {
		progress,
		verifyProgress,
		conflicts,
		skippedConflicts,
		startFull,
		startIncremental,
		cancel,
		refreshConflicts,
		clearSkippedConflicts,
	};
}

export type SyncState = ReturnType<typeof useSync>;
