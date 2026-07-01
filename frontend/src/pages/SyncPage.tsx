import { useState, useEffect } from "react";
import {
	Play,
	RefreshCw,
	Square,
	AlertCircle,
	Eye,
	ShieldAlert,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { SyncProgressBar } from "@/components/ProgressBar";
import { useSyncContext } from "@/hooks/SyncProvider";
import * as api from "@/api/client";
import type { SyncRun, DiffEntry, Conflict } from "@/api/types";
import { formatBytes } from "@/lib/utils";

export function SyncPage() {
	const {
		progress,
		conflicts,
		skippedConflicts,
		startFull,
		startIncremental,
		cancel,
		refreshConflicts,
		clearSkippedConflicts,
	} = useSyncContext();
	const [history, setHistory] = useState<SyncRun[]>([]);
	const [error, setError] = useState<string | null>(null);
	const [diff, setDiff] = useState<DiffEntry[] | null>(null);
	const [diffLoading, setDiffLoading] = useState(false);
	const [conflictsLoading, setConflictsLoading] = useState(false);

	useEffect(() => {
		api
			.getSyncHistory()
			.then((runs) => setHistory(runs ?? []))
			.catch(() => {});
	}, [progress.is_running]);

	const handleSync = async (mode: "full" | "incremental") => {
		try {
			setError(null);
			setDiff(null);
			if (mode === "full") await startFull();
			else await startIncremental();
		} catch (e) {
			setError(e instanceof Error ? e.message : "Failed to start sync");
		}
	};

	const handleDiff = async () => {
		try {
			setDiffLoading(true);
			setError(null);
			const entries = await api.getSyncDiff();
			setDiff(entries ?? []);
		} catch (e) {
			setError(e instanceof Error ? e.message : "Failed to get diff");
		} finally {
			setDiffLoading(false);
		}
	};

	const handleRefreshConflicts = async () => {
		try {
			setConflictsLoading(true);
			setError(null);
			await refreshConflicts();
			clearSkippedConflicts();
		} catch (e) {
			setError(e instanceof Error ? e.message : "Failed to check conflicts");
		} finally {
			setConflictsLoading(false);
		}
	};

	const renderConflictType = (c: Conflict) =>
		c.type === "modified_locally" ? "Modified locally" : "Missing locally";

	return (
		<div className="space-y-6">
			<Card>
				<CardHeader>
					<CardTitle className="flex items-center justify-between">
						<span>Sync</span>
						{progress.is_running && <Badge variant="default">Running</Badge>}
					</CardTitle>
				</CardHeader>
				<CardContent className="space-y-4">
					<div className="flex gap-2">
						<Button
							onClick={() => handleSync("full")}
							disabled={progress.is_running}
						>
							<Play className="mr-2 h-4 w-4" />
							Full Sync
						</Button>
						<Button
							variant="secondary"
							onClick={() => handleSync("incremental")}
							disabled={progress.is_running}
						>
							<RefreshCw className="mr-2 h-4 w-4" />
							Incremental
						</Button>
						<Button
							variant="outline"
							onClick={handleDiff}
							disabled={progress.is_running || diffLoading}
						>
							<Eye className="mr-2 h-4 w-4" />
							{diffLoading ? "Checking..." : "Preview"}
						</Button>
						{progress.is_running && (
							<Button variant="destructive" onClick={cancel}>
								<Square className="mr-2 h-4 w-4" />
								Cancel
							</Button>
						)}
						<Button
							variant="outline"
							onClick={handleRefreshConflicts}
							disabled={progress.is_running || conflictsLoading}
						>
							<ShieldAlert className="mr-2 h-4 w-4" />
							{conflictsLoading ? "Checking..." : "Check Conflicts"}
						</Button>
					</div>

					{error && (
						<div className="flex items-center gap-2 text-sm text-destructive">
							<AlertCircle className="h-4 w-4" />
							{error}
						</div>
					)}

					{progress.is_running && <SyncProgressBar progress={progress} />}

					{progress.errors.length > 0 && (
						<ScrollArea className="h-32 rounded border p-3">
							{progress.errors.map((err, i) => (
								<div key={i} className="text-sm text-destructive">
									{err}
								</div>
							))}
						</ScrollArea>
					)}
				</CardContent>
			</Card>

			{/* Conflicts */}
			{(conflicts.length > 0 || skippedConflicts.length > 0) && (
				<Card>
					<CardHeader>
						<CardTitle className="flex items-center justify-between">
							<span className="flex items-center gap-2">
								<ShieldAlert className="h-5 w-5 text-destructive" />
								Conflicts
							</span>
							<div className="flex items-center gap-2">
								<Badge variant="destructive">{conflicts.length} detected</Badge>
								{skippedConflicts.length > 0 && (
									<Badge variant="secondary">
										{skippedConflicts.length} skipped
									</Badge>
								)}
							</div>
						</CardTitle>
					</CardHeader>
					<CardContent>
						<p className="text-sm text-muted-foreground mb-3">
							Local files changed or disappeared since the last sync. Default
							strategy is &quot;skip&quot;; set{" "}
							<code>conflict_strategy: &quot;overwrite&quot;</code> to override.
						</p>
						<ScrollArea viewportClassName="max-h-60">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>File</TableHead>
										<TableHead>Type</TableHead>
										<TableHead>Path</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{conflicts.map((c) => (
										<TableRow key={c.file_id}>
											<TableCell
												className="font-medium truncate max-w-48"
												title={c.name}
											>
												{c.name}
											</TableCell>
											<TableCell>
												<Badge
													variant={
														c.type === "missing_locally"
															? "destructive"
															: "secondary"
													}
												>
													{renderConflictType(c)}
												</Badge>
											</TableCell>
											<TableCell
												className="text-muted-foreground text-sm truncate max-w-64"
												title={c.local_path}
											>
												{c.local_path}
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</ScrollArea>
					</CardContent>
				</Card>
			)}

			{/* Diff Preview */}
			{diff !== null && (
				<Card>
					<CardHeader>
						<CardTitle className="flex items-center justify-between">
							<span>Preview</span>
							<div className="flex items-center gap-2">
								<Badge variant="secondary">{diff.length} changes</Badge>
								<Button size="sm" variant="ghost" onClick={() => setDiff(null)}>
									Dismiss
								</Button>
							</div>
						</CardTitle>
					</CardHeader>
					<CardContent>
						{diff.length === 0 ? (
							<p className="text-sm text-muted-foreground text-center py-4">
								Everything is up to date
							</p>
						) : (
							<ScrollArea viewportClassName="max-h-80">
								<Table>
									<TableHeader>
										<TableRow>
											<TableHead>Action</TableHead>
											<TableHead>Name</TableHead>
											<TableHead className="text-right">Size</TableHead>
											<TableHead>Modified</TableHead>
										</TableRow>
									</TableHeader>
									<TableBody>
										{diff.map((entry) => (
											<TableRow key={entry.file_id}>
												<TableCell>
													<Badge
														variant={
															entry.action === "new"
																? "default"
																: entry.action === "delete"
																	? "destructive"
																	: "secondary"
														}
														className="text-xs"
													>
														{entry.action}
													</Badge>
												</TableCell>
												<TableCell
													className="font-medium truncate max-w-48"
													title={entry.name}
												>
													{entry.name}
												</TableCell>
												<TableCell className="text-right">
													{formatBytes(entry.size)}
												</TableCell>
												<TableCell className="text-muted-foreground text-sm">
													{new Date(entry.drive_modified).toLocaleString()}
												</TableCell>
											</TableRow>
										))}
									</TableBody>
								</Table>
							</ScrollArea>
						)}
					</CardContent>
				</Card>
			)}

			<Card>
				<CardHeader>
					<CardTitle>History</CardTitle>
				</CardHeader>
				<CardContent>
					{history.length === 0 ? (
						<p className="text-sm text-muted-foreground">No sync runs yet</p>
					) : (
						<div className="space-y-2">
							{history.map((run) => (
								<div
									key={run.id}
									className="flex items-center justify-between rounded border p-3 text-sm"
								>
									<div className="flex items-center gap-3">
										<Badge
											variant={
												run.status === "completed"
													? "default"
													: run.status === "failed"
														? "destructive"
														: "secondary"
											}
										>
											{run.status}
										</Badge>
										<span className="text-muted-foreground">
											{new Date(run.started_at).toLocaleString()}
										</span>
									</div>
									<span>
										{run.files_synced} synced, {run.files_failed} failed
									</span>
								</div>
							))}
						</div>
					)}
				</CardContent>
			</Card>
		</div>
	);
}
