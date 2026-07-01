import { useState, useEffect, useRef } from "react";
import {
	RefreshCw,
	Trash2,
	AlertCircle,
	CheckCircle,
	Clock,
	XCircle,
	Loader2,
	Play,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useSyncContext } from "@/hooks/SyncProvider";
import * as api from "@/api/client";
import type { Conversion, ConverterInfo, SyncedFile } from "@/api/types";
import { globToRegex } from "@/lib/utils";

export function ConversionsPage() {
	const { convertJobs } = useSyncContext();
	const [history, setHistory] = useState<Conversion[]>([]);
	const [loading, setLoading] = useState(true);
	const [stale, setStale] = useState<Conversion[]>([]);
	const [converters, setConverters] = useState<ConverterInfo[]>([]);
	const [allFiles, setAllFiles] = useState<SyncedFile[]>([]);
	const [queueing, setQueueing] = useState<string | null>(null);
	const loadedRef = useRef(false);

	const fetchAll = () => {
		setLoading(true);
		Promise.all([
			api.listConversions(50),
			api.listStaleConversions(),
			api.listConverters(),
			api.listFiles(""),
		])
			.then(([convs, stales, convList, files]) => {
				setHistory(convs ?? []);
				setStale(stales ?? []);
				setConverters((convList ?? []).filter((c) => c.enabled));
				setAllFiles(files ?? []);
			})
			.catch(() => {})
			.finally(() => setLoading(false));
	};

	useEffect(() => {
		if (loadedRef.current) return;
		loadedRef.current = true;
		requestAnimationFrame(() => fetchAll());
	}, []);

	const handleReconvert = async (fileId: string, converter: string) => {
		try {
			await api.reconvertFile(fileId, converter);
			toast.success(`Re-queued ${converter} conversion`);
			setTimeout(fetchAll, 1000);
		} catch (e) {
			toast.error("Reconvert failed", {
				description: e instanceof Error ? e.message : undefined,
			});
		}
	};

	const handleDelete = async (id: string) => {
		try {
			await api.deleteConversion(id);
			toast.success("Conversion record deleted");
			fetchAll();
		} catch {
			toast.error("Delete failed");
		}
	};

	const handleConvertAll = async (converter: ConverterInfo) => {
		const re = globToRegex(converter.input_pattern);
		const matched = allFiles.filter((f) => !f.is_folder && re.test(f.name));
		if (matched.length === 0) {
			toast.error(`No files match ${converter.input_pattern}`);
			return;
		}

		setQueueing(converter.name);
		let ok = 0;
		let fail = 0;
		for (const f of matched) {
			try {
				await api.convertFile(f.file_id, converter.name);
				ok++;
			} catch {
				fail++;
			}
		}
		if (ok > 0) {
			toast.success(
				`Queued ${ok} file(s) for ${converter.name}${fail > 0 ? ` (${fail} failed)` : ""}`,
			);
		}
		setQueueing(null);
		setTimeout(fetchAll, 1000);
	};

	const activeJobs = Object.entries(convertJobs);
	const staleIds = new Set(stale.map((s) => s.id));

	const statusIcon = (status: string) => {
		switch (status) {
			case "completed":
				return <CheckCircle className="h-4 w-4 text-green-500" />;
			case "failed":
				return <XCircle className="h-4 w-4 text-destructive" />;
			case "running":
				return <Loader2 className="h-4 w-4 animate-spin text-blue-500" />;
			default:
				return <Clock className="h-4 w-4 text-muted-foreground" />;
		}
	};

	const formatPath = (path: string) => {
		const parts = path.split("/");
		if (parts.length > 3) {
			return "…/" + parts.slice(-3).join("/");
		}
		return path;
	};

	return (
		<div className="space-y-6">
			{/* Converters — batch convert */}
			{converters.length > 0 && (
				<Card>
					<CardHeader>
						<CardTitle>Converters</CardTitle>
					</CardHeader>
					<CardContent className="space-y-2">
						{converters.map((c) => {
							const re = globToRegex(c.input_pattern);
							const count = allFiles.filter(
								(f) => !f.is_folder && re.test(f.name),
							).length;
							return (
								<div
									key={c.name}
									className="flex items-center justify-between rounded border p-3 text-sm"
								>
									<div className="flex items-center gap-3">
										<span className="font-medium">{c.name}</span>
										<Badge variant="outline" className="font-mono text-xs">
											{c.input_pattern}
										</Badge>
										<span className="text-muted-foreground">
											{count} file{count !== 1 ? "s" : ""} match
										</span>
									</div>
									<Button
										size="sm"
										onClick={() => handleConvertAll(c)}
										disabled={queueing === c.name || count === 0}
									>
										{queueing === c.name ? (
											<Loader2 className="mr-2 h-4 w-4 animate-spin" />
										) : (
											<Play className="mr-2 h-4 w-4" />
										)}
										Convert All
									</Button>
								</div>
							);
						})}
					</CardContent>
				</Card>
			)}

			{/* Active conversion jobs */}
			{activeJobs.length > 0 && (
				<Card>
					<CardHeader>
						<CardTitle className="flex items-center gap-2">
							<Loader2 className="h-5 w-5 animate-spin" />
							Active Conversions
							<Badge variant="default">{activeJobs.length}</Badge>
						</CardTitle>
					</CardHeader>
					<CardContent className="space-y-3">
						{activeJobs.map(([jobId, job]) => (
							<div key={jobId} className="space-y-1">
								<div className="flex items-center justify-between text-sm">
									<span className="font-medium">{job.file_id}</span>
									<span className="text-muted-foreground">{job.converter}</span>
									<span className="font-mono text-xs">
										{Math.round(job.progress * 100)}%
									</span>
								</div>
								<Progress value={job.progress * 100} className="h-2" />
							</div>
						))}
					</CardContent>
				</Card>
			)}

			{/* Stale conversions */}
			{stale.length > 0 && (
				<Card className="border-destructive/30">
					<CardHeader>
						<CardTitle className="flex items-center gap-2">
							<AlertCircle className="h-5 w-5 text-destructive" />
							Stale Conversions
							<Badge variant="destructive">{stale.length}</Badge>
						</CardTitle>
					</CardHeader>
					<CardContent>
						<p className="text-sm text-muted-foreground mb-3">
							Original files have changed since these conversions were run.
						</p>
						<div className="space-y-2">
							{stale.map((c) => (
								<div
									key={c.id}
									className="flex items-center justify-between rounded border p-2 text-sm"
								>
									<div className="flex items-center gap-2 min-w-0">
										<span className="truncate font-medium">{c.file_id}</span>
										<Badge variant="outline" className="shrink-0">
											{c.converter}
										</Badge>
									</div>
									<div className="flex items-center gap-1 shrink-0">
										<Button
											size="sm"
											variant="outline"
											onClick={() => handleReconvert(c.file_id, c.converter)}
										>
											<RefreshCw className="mr-1 h-3 w-3" />
											Re-run
										</Button>
										<Button
											size="sm"
											variant="ghost"
											onClick={() => handleDelete(c.id)}
										>
											<Trash2 className="h-3 w-3" />
										</Button>
									</div>
								</div>
							))}
						</div>
					</CardContent>
				</Card>
			)}

			{/* Conversion history */}
			<Card>
				<CardHeader className="flex flex-row items-center justify-between">
					<CardTitle>History</CardTitle>
					<Button
						size="sm"
						variant="outline"
						onClick={fetchAll}
						disabled={loading}
					>
						<RefreshCw
							className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`}
						/>
						Refresh
					</Button>
				</CardHeader>
				<CardContent>
					{loading ? (
						<p className="text-sm text-muted-foreground py-8 text-center">
							Loading...
						</p>
					) : history.length === 0 ? (
						<p className="text-sm text-muted-foreground py-8 text-center">
							No conversion jobs yet. Use the Convert All button above, or
							select files in the Files tab and click a converter.
						</p>
					) : (
						<ScrollArea viewportClassName="max-h-96">
							<div className="space-y-2">
								{history.map((c) => (
									<div
										key={c.id}
										className={`flex items-center justify-between rounded border p-3 text-sm ${
											staleIds.has(c.id)
												? "border-destructive/30 bg-destructive/5"
												: ""
										}`}
									>
										<div className="flex items-center gap-3 min-w-0">
											{statusIcon(c.status)}
											<div className="min-w-0">
												<div className="font-medium truncate" title={c.file_id}>
													{c.file_id}
												</div>
												<div className="text-xs text-muted-foreground flex gap-2">
													<span>{c.converter}</span>
													{c.output_path && (
														<span className="truncate" title={c.output_path}>
															→ {formatPath(c.output_path)}
														</span>
													)}
												</div>
											</div>
										</div>
										<div className="flex items-center gap-2 shrink-0">
											<Badge
												variant={
													c.status === "completed"
														? "default"
														: c.status === "failed"
															? "destructive"
															: "secondary"
												}
											>
												{c.status}
											</Badge>
											{c.status === "failed" && c.error_message && (
												<span
													className="text-xs text-destructive max-w-32 truncate"
													title={c.error_message}
												>
													{c.error_message}
												</span>
											)}
											{c.started_at && (
												<span className="text-xs text-muted-foreground">
													{new Date(c.started_at).toLocaleString()}
												</span>
											)}
											<div className="flex gap-1">
												{c.status !== "running" && (
													<Button
														size="sm"
														variant="ghost"
														onClick={() =>
															handleReconvert(c.file_id, c.converter)
														}
													>
														<RefreshCw className="h-3 w-3" />
													</Button>
												)}
												<Button
													size="sm"
													variant="ghost"
													onClick={() => handleDelete(c.id)}
												>
													<Trash2 className="h-3 w-3" />
												</Button>
											</div>
										</div>
									</div>
								))}
							</div>
						</ScrollArea>
					)}
				</CardContent>
			</Card>
		</div>
	);
}
