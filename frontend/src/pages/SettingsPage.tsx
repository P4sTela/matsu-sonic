import { useState, useEffect } from "react";
import { Save, CheckCircle, AlertCircle, FolderOpen, Trash2, Plus, X } from "lucide-react";
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

  if (loading) return <p className="text-center text-muted-foreground py-8">Loading...</p>;
  if (!draft) return <p className="text-center text-destructive py-8">{error}</p>;

  return (
    <div className="space-y-6">
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
              <Button variant="outline" size="sm" onClick={() => setCredsBrowserOpen(true)}>
                <FolderOpen className="h-4 w-4" />
              </Button>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="secondary" onClick={handleTestAuth}>
              Test Auth
            </Button>
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
                title={authStatus !== "ok" ? "Authenticate first to browse Drive" : "Browse Drive folders"}
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
              <Button variant="outline" size="sm" onClick={() => setSyncDirBrowserOpen(true)}>
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
                onChange={(e) => update({ max_workers: Number(e.target.value) })}
                min={1}
                max={10}
              />
            </div>
            <div>
              <Label>Chunk Size (MB)</Label>
              <Input
                type="number"
                value={draft.chunk_size_mb}
                onChange={(e) => update({ chunk_size_mb: Number(e.target.value) })}
                min={1}
                max={100}
              />
            </div>
          </div>
          <Button onClick={handleSave} disabled={!dirty}>
            <Save className="mr-2 h-4 w-4" />
            {dirty ? "Save" : "Saved"}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Ignore Patterns</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-xs text-muted-foreground">
            Glob patterns to exclude from sync (e.g. <code>*.mp4</code>, <code>backup_*</code>). Matched against file names.
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
                    const next = (draft.ignore_patterns ?? []).filter((_, j) => j !== i);
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
            onClick={() => update({ ignore_patterns: [...(draft.ignore_patterns ?? []), ""] })}
          >
            <Plus className="mr-2 h-4 w-4" />
            Add Pattern
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Danger Zone</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          <p className="text-sm text-muted-foreground">
            Reset all sync data (file records, run history). Local files are not deleted.
          </p>
          <Button
            variant="destructive"
            onClick={async () => {
              if (!window.confirm("Reset all sync data? This cannot be undone.")) return;
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
