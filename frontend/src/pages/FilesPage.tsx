import { useState, useEffect, useCallback } from "react";
import { Search, Trash2, Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import * as api from "@/api/client";
import type { SyncedFile } from "@/api/types";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "—";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

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

  const toggleAll = () => {
    if (selectedIds.size === files.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(files.map((f) => f.file_id)));
    }
  };

  const handleDelete = async () => {
    if (selectedIds.size === 0) return;
    if (!window.confirm(`Remove ${selectedIds.size} file record(s) from database?`)) return;
    await api.deleteFiles([...selectedIds]);
    setSelectedIds(new Set());
    load();
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span>Synced Files</span>
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
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-10">
                  <button onClick={toggleAll} className="flex items-center justify-center">
                    <div className={`h-4 w-4 rounded border flex items-center justify-center ${
                      files.length > 0 && selectedIds.size === files.length
                        ? "bg-primary border-primary" : "border-input"
                    }`}>
                      {files.length > 0 && selectedIds.size === files.length && (
                        <Check className="h-3 w-3 text-primary-foreground" />
                      )}
                    </div>
                  </button>
                </TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Type</TableHead>
                <TableHead className="text-right">Size</TableHead>
                <TableHead>Last Synced</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(files ?? []).map((f) => (
                <TableRow
                  key={f.file_id}
                  className="cursor-pointer"
                  onClick={() => toggleSelect(f.file_id)}
                >
                  <TableCell>
                    <div className={`h-4 w-4 rounded border flex items-center justify-center ${
                      selectedIds.has(f.file_id) ? "bg-primary border-primary" : "border-input"
                    }`}>
                      {selectedIds.has(f.file_id) && <Check className="h-3 w-3 text-primary-foreground" />}
                    </div>
                  </TableCell>
                  <TableCell className="font-medium">{f.name}</TableCell>
                  <TableCell>
                    <Badge variant="secondary" className="text-xs">
                      {f.is_folder ? "Folder" : f.mime_type.split("/").pop()}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">{formatBytes(f.size)}</TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {f.last_synced ? new Date(f.last_synced).toLocaleString() : "—"}
                  </TableCell>
                </TableRow>
              ))}
              {files.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                    No files synced yet
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
