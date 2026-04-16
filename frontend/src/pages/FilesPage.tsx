import { useState, useEffect, useCallback, useMemo } from "react";
import { Search, Trash2, ShieldCheck, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { FileTreePicker } from "@/components/FileTreePicker";
import * as api from "@/api/client";
import type { SyncedFile } from "@/api/types";

export function FilesPage() {
  const [files, setFiles] = useState<SyncedFile[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const load = useCallback(() => {
    setLoading(true);
    api
      .listFiles(search)
      .then((f) => setFiles(f ?? []))
      .catch(() => setFiles([]))
      .finally(() => setLoading(false));
  }, [search]);

  useEffect(() => {
    const timer = setTimeout(load, 300);
    return () => clearTimeout(timer);
  }, [load]);

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleDelete = async () => {
    if (selectedIds.size === 0) return;
    if (!window.confirm(`Remove ${selectedIds.size} file record(s) from database?`)) return;
    await api.deleteFiles([...selectedIds]);
    setSelectedIds(new Set());
    load();
  };

  const folderCount = useMemo(() => files.filter(f => f.is_folder).length, [files]);
  const fileCount = useMemo(() => files.filter(f => !f.is_folder).length, [files]);

  const [verifying, setVerifying] = useState(false);
  const [badFileIds, setBadFileIds] = useState<string[]>([]);
  const [resyncing, setResyncing] = useState(false);

  const handleVerify = async () => {
    setVerifying(true);
    setBadFileIds([]);
    try {
      const res = await api.verifyFiles();
      const problems = res.mismatch + res.missing;
      const ids = res.results
        .filter((r) => r.status === "mismatch" || r.status === "missing")
        .map((r) => r.file_id);
      setBadFileIds(ids);
      if (problems > 0) {
        toast.error(`Verified ${res.total} files: ${res.ok} ok, ${res.mismatch} mismatch, ${res.missing} missing`);
      } else {
        toast.success(`All ${res.ok} files verified OK${res.skipped > 0 ? ` (${res.skipped} skipped)` : ""}`);
      }
    } catch (e) {
      toast.error("Verification failed", {
        description: e instanceof Error ? e.message : undefined,
      });
    } finally {
      setVerifying(false);
    }
  };

  const handleResync = async () => {
    if (badFileIds.length === 0) return;
    setResyncing(true);
    try {
      await api.resyncFiles(badFileIds);
      await api.startFullSync();
      setBadFileIds([]);
      toast.success(`Re-syncing ${badFileIds.length} files`);
    } catch (e) {
      toast.error("Re-sync failed", {
        description: e instanceof Error ? e.message : undefined,
      });
    } finally {
      setResyncing(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span>Synced Files</span>
            <span className="text-sm font-normal text-muted-foreground">
              {folderCount > 0 && `${folderCount} folders, `}{fileCount} files
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Button size="sm" variant="outline" onClick={handleVerify} disabled={verifying}>
              <ShieldCheck className="mr-2 h-4 w-4" />
              {verifying ? "Verifying..." : "Verify"}
            </Button>
            {badFileIds.length > 0 && (
              <Button size="sm" variant="outline" onClick={handleResync} disabled={resyncing}>
                <RefreshCw className={`mr-2 h-4 w-4 ${resyncing ? "animate-spin" : ""}`} />
                Re-sync {badFileIds.length} files
              </Button>
            )}
            {selectedIds.size > 0 && (
              <>
                <Badge variant="secondary">{selectedIds.size} selected</Badge>
                <Button size="sm" variant="destructive" onClick={handleDelete}>
                  <Trash2 className="mr-2 h-4 w-4" />
                  Remove
                </Button>
              </>
            )}
          </div>
        </CardTitle>
        <div className="relative">
          <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search files..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
      </CardHeader>
      <CardContent>
        {loading ? (
          <p className="text-sm text-muted-foreground py-8 text-center">Loading...</p>
        ) : (
          <FileTreePicker
            files={files}
            selectedIds={selectedIds}
            onToggleSelect={toggleSelect}
          />
        )}
      </CardContent>
    </Card>
  );
}
