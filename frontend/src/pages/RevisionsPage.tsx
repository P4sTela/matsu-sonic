import { useState, useEffect, useMemo } from "react";
import { Download, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { FileTreePicker } from "@/components/FileTreePicker";
import * as api from "@/api/client";
import type { SyncedFile, DriveRevision } from "@/api/types";
import { formatBytes } from "@/lib/utils";

export function RevisionsPage() {
  const [files, setFiles] = useState<SyncedFile[]>([]);
  const [search, setSearch] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [revisions, setRevisions] = useState<DriveRevision[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => {
      api.listFiles(search).then((f) => setFiles(f ?? [])).catch(() => {});
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const selectedFile = useMemo(
    () => files.find((f) => f.file_id === selectedId) ?? null,
    [files, selectedId],
  );

  const selectedIds = useMemo(
    () => (selectedId ? new Set([selectedId]) : new Set<string>()),
    [selectedId],
  );

  useEffect(() => {
    if (!selectedFile) return;
    setLoading(true);
    api
      .listRevisions(selectedFile.file_id)
      .then(setRevisions)
      .catch(() => setRevisions([]))
      .finally(() => setLoading(false));
  }, [selectedFile]);

  const handleDownload = async (revId: string) => {
    if (!selectedFile) return;
    try {
      await api.downloadRevision(selectedFile.file_id, revId);
    } catch {
      // TODO: show error
    }
  };

  const handleToggleSelect = (id: string) => {
    setSelectedId((prev) => (prev === id ? null : id));
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Select File</CardTitle>
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
          <FileTreePicker
            files={files}
            selectedIds={selectedIds}
            onToggleSelect={handleToggleSelect}
            showDetails={false}
            showCheckbox={false}
            maxHeightClass="max-h-72"
          />
        </CardContent>
      </Card>

      {selectedFile && (
        <Card>
          <CardHeader>
            <CardTitle>Revisions: {selectedFile.name}</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <p className="text-sm text-muted-foreground text-center py-4">Loading...</p>
            ) : revisions.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-4">No revisions found</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Modified</TableHead>
                    <TableHead>User</TableHead>
                    <TableHead className="text-right">Size</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {revisions.map((rev) => (
                    <TableRow key={rev.id}>
                      <TableCell className="text-sm">
                        {new Date(rev.modifiedTime).toLocaleString()}
                      </TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {rev.lastModifyingUser?.displayName || "—"}
                      </TableCell>
                      <TableCell className="text-right text-sm">
                        {rev.size ? formatBytes(Number(rev.size)) : "—"}
                      </TableCell>
                      <TableCell>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => handleDownload(rev.id)}
                        >
                          <Download className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
