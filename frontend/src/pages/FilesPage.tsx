import { useState, useEffect, useCallback, useMemo } from "react";
import { Search, Trash2 } from "lucide-react";
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
          {selectedIds.size > 0 && (
            <div className="flex items-center gap-2">
              <Badge variant="secondary">{selectedIds.size} selected</Badge>
              <Button size="sm" variant="destructive" onClick={handleDelete}>
                <Trash2 className="mr-2 h-4 w-4" />
                Remove
              </Button>
            </div>
          )}
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
