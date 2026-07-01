import { useState, useEffect } from "react";
import {
	Save,
	CheckCircle,
	AlertCircle,
	FolderOpen,
	Trash2,
	Plus,
	X,
	Zap,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { DirBrowser } from "@/components/DirBrowser";
import { DriveBrowser } from "@/components/DriveBrowser";
import { useConfig } from "@/hooks/useConfig";
import type { Config } from "@/api/types";
import * as api from "@/api/client";

export function SettingsPage() {
	const { config, loading, error, save } = useConfig();
	const [draft, setDraft] = useState<Config | null>(null);
	const [dirty, setDirty] = useState(false);
	const [authStatus, setAuthStatus] = useState<string | null>(null);
	const [authUser, setAuthUser] = useState<string | null>(null);
	const [authorizing, setAuthorizing] = useState(false);
	const [preview, setPreview] = useState<{
		matched: number;
		total: number;
		samples: string[];
	} | null>(null);
	const [previewing, setPreviewing] = useState(false);
	const [credsBrowserOpen, setCredsBrowserOpen] = useState(false);
	const [syncDirBrowserOpen, setSyncDirBrowserOpen] = useState(false);
	const [driveBrowserOpen, setDriveBrowserOpen] = useState(false);

	useEffect(() => {
		if (config) {
			setDraft(config);
			setDirty(false);
		}
	}, [config]);

	const update = (partial: Partial<Config>) => {
		setDraft((prev) => (prev ? { ...prev, ...partial } : prev));
		setDirty(true);
	};

	const handleSave = async () => {
		if (!draft) return;
		await save(draft);
		setDirty(false);
	};

	const handleTestAuth = async () => {
		try {
			setAuthStatus("testing");
			const result = await api.testAuth();
			setAuthStatus("ok");
			setAuthUser(`${result.user.displayName} (${result.user.emailAddress})`);
		} catch (e) {
			setAuthStatus("error");
			setAuthUser(e instanceof Error ? e.message : "Auth failed");
		}
	};

	const handleAuthorize = async () => {
		setAuthorizing(true);
		setAuthStatus("testing");
		try {
			const { auth_url } = await api.startAuth();
			window.open(auth_url, "_blank", "noopener,noreferrer");
			// Poll until the browser callback completes the exchange.
			for (let i = 0; i < 90; i++) {
				await new Promise((res) => setTimeout(res, 2000));
				try {
					const result = await api.testAuth();
					setAuthStatus("ok");
					setAuthUser(
						`${result.user.displayName} (${result.user.emailAddress})`,
					);
					return;
				} catch {
					// not ready yet
				}
			}
			setAuthStatus("error");
			setAuthUser("Authorization not completed (timed out)");
		} catch (e) {
			setAuthStatus("error");
			setAuthUser(e instanceof Error ? e.message : "Authorization failed");
		} finally {
			setAuthorizing(false);
		}
	};

	const handlePreview = async () => {
		if (!draft) return;
		setPreviewing(true);
		try {
			setPreview(await api.previewSelect(draft.select_patterns ?? []));
		} catch {
			setPreview(null);
		} finally {
			setPreviewing(false);
		}
	};

	if (loading)
		return <p className="text-center text-muted-foreground py-8">Loading...</p>;
	if (!draft)
		return <p className="text-center text-destructive py-8">{error}</p>;

	return (
		<div className="space-y-6">
			{/* Sticky save bar */}
			{dirty && (
				<div className="sticky top-0 z-10 -mx-4 px-4 py-3 bg-background/95 backdrop-blur border-b flex items-center justify-between">
					<span className="text-sm text-muted-foreground">Unsaved changes</span>
					<Button onClick={handleSave}>
						<Save className="mr-2 h-4 w-4" />
						Save Settings
					</Button>
				</div>
			)}

			<Card>
				<CardHeader>
					<CardTitle>Authentication</CardTitle>
				</CardHeader>
				<CardContent className="space-y-4">
					<div>
						<Label>Auth Method</Label>
						<Input value={draft.auth_method} disabled />
					</div>
					<div>
						<Label>Credentials Path</Label>
						<div className="flex gap-2">
							<Input
								value={draft.credentials_path}
								onChange={(e) => update({ credentials_path: e.target.value })}
								className="flex-1"
							/>
							<Button
								variant="outline"
								size="sm"
								onClick={() => setCredsBrowserOpen(true)}
							>
								<FolderOpen className="h-4 w-4" />
							</Button>
						</div>
					</div>
					<div className="flex items-center gap-2 flex-wrap">
						<Button variant="secondary" onClick={handleTestAuth}>
							Test Auth
						</Button>
						{draft.auth_method !== "service_account" && (
							<Button
								variant="outline"
								onClick={handleAuthorize}
								disabled={authorizing}
							>
								{authorizing
									? "Waiting for browser…"
									: "Authorize / Re-authorize"}
							</Button>
						)}
						{authorizing && (
							<span className="text-xs text-muted-foreground">
								A browser tab was opened. Approve access, then return here.
							</span>
						)}
						{authStatus === "ok" && (
							<span className="flex items-center gap-1 text-sm text-green-600">
								<CheckCircle className="h-4 w-4" />
								{authUser}
							</span>
						)}
						{authStatus === "error" && (
							<span className="flex items-center gap-1 text-sm text-destructive">
								<AlertCircle className="h-4 w-4" />
								{authUser}
							</span>
						)}
					</div>
				</CardContent>
			</Card>

			<Card>
				<CardHeader>
					<CardTitle>Sync Settings</CardTitle>
				</CardHeader>
				<CardContent className="space-y-4">
					<div>
						<Label>Sync Folder ID</Label>
						<div className="flex gap-2">
							<Input
								value={draft.sync_folder_id}
								onChange={(e) => update({ sync_folder_id: e.target.value })}
								placeholder="Google Drive Folder ID"
								className="flex-1"
							/>
							<Button
								variant="outline"
								size="sm"
								onClick={() => setDriveBrowserOpen(true)}
								disabled={authStatus !== "ok"}
								title={
									authStatus !== "ok"
										? "Authenticate first to browse Drive"
										: "Browse Drive folders"
								}
							>
								<FolderOpen className="h-4 w-4" />
							</Button>
						</div>
						<p className="text-xs text-muted-foreground mt-1">
							{draft.auth_method === "service_account"
								? "Share the folder with the service account email, then paste the folder ID from the URL (drive.google.com/drive/folders/<ID>)"
								: "Authenticate first, then use the browse button to select a folder"}
						</p>
					</div>
					<div>
						<Label>Local Sync Directory</Label>
						<div className="flex gap-2">
							<Input
								value={draft.local_sync_dir}
								onChange={(e) => update({ local_sync_dir: e.target.value })}
								placeholder="/path/to/sync"
								className="flex-1"
							/>
							<Button
								variant="outline"
								size="sm"
								onClick={() => setSyncDirBrowserOpen(true)}
							>
								<FolderOpen className="h-4 w-4" />
							</Button>
						</div>
					</div>
					<Separator />
					<div className="grid grid-cols-2 gap-4">
						<div>
							<Label>Max Workers</Label>
							<Input
								type="number"
								value={draft.max_workers}
								onChange={(e) =>
									update({ max_workers: Number(e.target.value) })
								}
								min={1}
								max={10}
							/>
						</div>
						<div>
							<Label>Chunk Size (MB)</Label>
							<Input
								type="number"
								value={draft.chunk_size_mb}
								onChange={(e) =>
									update({ chunk_size_mb: Number(e.target.value) })
								}
								min={1}
								max={100}
							/>
						</div>
					</div>
				</CardContent>
			</Card>

			<Card>
				<CardHeader>
					<CardTitle>Ignore Patterns</CardTitle>
				</CardHeader>
				<CardContent className="space-y-3">
					<p className="text-xs text-muted-foreground">
						Glob patterns to exclude from sync (e.g. <code>*.mp4</code>,{" "}
						<code>backup_*</code>). Matched against file names.
					</p>
					<div className="space-y-2">
						{(draft.ignore_patterns ?? []).map((pattern, i) => (
							<div key={i} className="flex items-center gap-2">
								<Input
									value={pattern}
									onChange={(e) => {
										const next = [...(draft.ignore_patterns ?? [])];
										next[i] = e.target.value;
										update({ ignore_patterns: next });
									}}
									className="flex-1 font-mono text-sm"
									placeholder="*.mp4"
								/>
								<Button
									size="sm"
									variant="ghost"
									onClick={() => {
										const next = (draft.ignore_patterns ?? []).filter(
											(_, j) => j !== i,
										);
										update({ ignore_patterns: next });
									}}
								>
									<X className="h-4 w-4" />
								</Button>
							</div>
						))}
					</div>
					<Button
						size="sm"
						variant="outline"
						onClick={() =>
							update({
								ignore_patterns: [...(draft.ignore_patterns ?? []), ""],
							})
						}
					>
						<Plus className="mr-2 h-4 w-4" />
						Add Pattern
					</Button>
				</CardContent>
			</Card>

			<Card>
				<CardHeader>
					<CardTitle>Selective Sync</CardTitle>
				</CardHeader>
				<CardContent className="space-y-3">
					<p className="text-xs text-muted-foreground">
						Include patterns to limit which files are synced. When empty,
						everything is synced. Matched against the file path relative to the
						sync root. A pattern without wildcards matches as a{" "}
						<strong>prefix</strong> (e.g. <code>videos/2024</code> matches
						everything under it). Use <code>*</code> for a single path segment
						and <code>**</code> for any depth (e.g. <code>videos/*</code>,{" "}
						<code>**/*.mp4</code>).
					</p>
					<div className="space-y-2">
						{(draft.select_patterns ?? []).map((pattern, i) => (
							<div key={i} className="flex items-center gap-2">
								<Input
									value={pattern}
									onChange={(e) => {
										const next = [...(draft.select_patterns ?? [])];
										next[i] = e.target.value;
										update({ select_patterns: next });
									}}
									className="flex-1 font-mono text-sm"
									placeholder="videos/2024"
								/>
								<Button
									size="sm"
									variant="ghost"
									onClick={() => {
										const next = (draft.select_patterns ?? []).filter(
											(_, j) => j !== i,
										);
										update({ select_patterns: next });
									}}
								>
									<X className="h-4 w-4" />
								</Button>
							</div>
						))}
					</div>
					<div className="flex items-center gap-2">
						<Button
							size="sm"
							variant="outline"
							onClick={() =>
								update({
									select_patterns: [...(draft.select_patterns ?? []), ""],
								})
							}
						>
							<Plus className="mr-2 h-4 w-4" />
							Add Pattern
						</Button>
						<Button
							size="sm"
							variant="secondary"
							onClick={handlePreview}
							disabled={previewing}
						>
							{previewing ? "Previewing…" : "Preview"}
						</Button>
					</div>
					{preview && (
						<div className="rounded border bg-muted/40 p-3 text-sm space-y-1">
							<p>
								<strong>{preview.matched}</strong> of{" "}
								<strong>{preview.total}</strong> synced files match.
							</p>
							{preview.samples.length > 0 && (
								<ul className="font-mono text-xs text-muted-foreground space-y-0.5">
									{preview.samples.map((s, i) => (
										<li key={i} className="truncate">
											{s}
										</li>
									))}
									{preview.matched > preview.samples.length && <li>…</li>}
								</ul>
							)}
							<p className="text-xs text-muted-foreground">
								Preview is based on already-synced files (no Drive request).
							</p>
						</div>
					)}
				</CardContent>
			</Card>

			<Card>
				<CardHeader>
					<CardTitle>Converters</CardTitle>
				</CardHeader>
				<CardContent className="space-y-3">
					<p className="text-xs text-muted-foreground">
						External command-based file converters (e.g. ffmpeg mp4→HAP). Use{" "}
						<code>{"{{input}}"}</code>, <code>{"{{output}}"}</code>,{" "}
						<code>{"{{stem}}"}</code> in commands.
					</p>
					<div className="space-y-4">
						{(draft.converters ?? []).map((c, i) => (
							<div key={i} className="rounded border p-3 space-y-2">
								<div className="flex items-center justify-between">
									<div className="flex items-center gap-2">
										<Zap className="h-4 w-4 text-muted-foreground" />
										<Input
											value={c.name}
											onChange={(e) => {
												const next = [...(draft.converters ?? [])];
												next[i] = { ...next[i], name: e.target.value };
												update({ converters: next });
											}}
											className="h-8 w-40 font-mono text-sm"
											placeholder="converter-name"
										/>
									</div>
									<div className="flex items-center gap-3">
										<label className="flex items-center gap-1 text-xs">
											<input
												type="checkbox"
												checked={c.enabled}
												onChange={(e) => {
													const next = [...(draft.converters ?? [])];
													next[i] = { ...next[i], enabled: e.target.checked };
													update({ converters: next });
												}}
												className="h-3 w-3"
											/>
											Enabled
										</label>
										<label className="flex items-center gap-1 text-xs">
											<input
												type="checkbox"
												checked={c.auto_convert}
												onChange={(e) => {
													const next = [...(draft.converters ?? [])];
													next[i] = {
														...next[i],
														auto_convert: e.target.checked,
													};
													update({ converters: next });
												}}
												className="h-3 w-3"
											/>
											Auto
										</label>
										<Button
											size="sm"
											variant="ghost"
											onClick={() => {
												const next = (draft.converters ?? []).filter(
													(_, j) => j !== i,
												);
												update({ converters: next });
											}}
										>
											<X className="h-4 w-4" />
										</Button>
									</div>
								</div>
								<div className="grid grid-cols-3 gap-2">
									<div>
										<Label className="text-xs">Input Pattern</Label>
										<Input
											value={c.input_pattern}
											onChange={(e) => {
												const next = [...(draft.converters ?? [])];
												next[i] = { ...next[i], input_pattern: e.target.value };
												update({ converters: next });
											}}
											className="h-8 font-mono text-sm"
											placeholder="*.mp4"
										/>
									</div>
									<div>
										<Label className="text-xs">Output Ext</Label>
										<Input
											value={c.output_extension}
											onChange={(e) => {
												const next = [...(draft.converters ?? [])];
												next[i] = {
													...next[i],
													output_extension: e.target.value,
												};
												update({ converters: next });
											}}
											className="h-8 font-mono text-sm"
											placeholder=".mov"
										/>
									</div>
									<div>
										<Label className="text-xs">Output Dir</Label>
										<Input
											value={c.output_dir}
											onChange={(e) => {
												const next = [...(draft.converters ?? [])];
												next[i] = { ...next[i], output_dir: e.target.value };
												update({ converters: next });
											}}
											className="h-8 font-mono text-sm"
											placeholder="converted/hap"
										/>
									</div>
								</div>
								<div>
									<Label className="text-xs">Command</Label>
									<Input
										value={c.command}
										onChange={(e) => {
											const next = [...(draft.converters ?? [])];
											next[i] = { ...next[i], command: e.target.value };
											update({ converters: next });
										}}
										className="h-8 font-mono text-sm"
										placeholder='ffmpeg -y -i {"{{input}}"} -c:v hap {"{{output}}"}'
									/>
								</div>
							</div>
						))}
					</div>
					<Button
						size="sm"
						variant="outline"
						onClick={() =>
							update({
								converters: [
									...(draft.converters ?? []),
									{
										name: "",
										enabled: true,
										input_pattern: "",
										output_extension: "",
										output_dir: "",
										command: "",
										auto_convert: false,
									},
								],
							})
						}
					>
						<Plus className="mr-2 h-4 w-4" />
						Add Converter
					</Button>
				</CardContent>
			</Card>

			<Card>
				<CardHeader>
					<CardTitle>Danger Zone</CardTitle>
				</CardHeader>
				<CardContent className="space-y-2">
					<p className="text-sm text-muted-foreground">
						Reset all sync data (file records, run history). Local files are not
						deleted.
					</p>
					<Button
						variant="destructive"
						onClick={async () => {
							if (
								!window.confirm("Reset all sync data? This cannot be undone.")
							)
								return;
							try {
								await api.resetSync();
								window.location.reload();
							} catch {
								// ignore
							}
						}}
					>
						<Trash2 className="mr-2 h-4 w-4" />
						Reset Sync Data
					</Button>
				</CardContent>
			</Card>

			<DirBrowser
				open={credsBrowserOpen}
				onOpenChange={setCredsBrowserOpen}
				onSelect={(path) => update({ credentials_path: path })}
				title="Select Credentials File"
				mode="file"
			/>
			<DirBrowser
				open={syncDirBrowserOpen}
				onOpenChange={setSyncDirBrowserOpen}
				onSelect={(path) => update({ local_sync_dir: path })}
				title="Select Sync Directory"
				mode="directory"
			/>
			<DriveBrowser
				open={driveBrowserOpen}
				onOpenChange={setDriveBrowserOpen}
				onSelect={(folderId) => update({ sync_folder_id: folderId })}
				onIgnore={(name) => {
					const current = draft?.ignore_patterns ?? [];
					if (!current.includes(name)) {
						update({ ignore_patterns: [...current, name] });
					}
				}}
				title="Select Drive Folder"
			/>
		</div>
	);
}
