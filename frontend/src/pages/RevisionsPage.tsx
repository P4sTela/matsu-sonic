import { useState, useEffect } from "react";
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
import * as api from "@/api/client";
import type { SyncedFile, DriveRevision } from "@/api/types";

export function RevisionsPage() {
  const [files, setFiles] = useState<SyncedFile[]>([]);
  const [search, setSearch] = useState("");
  const [selectedFile, setSelectedFile] = useState<SyncedFile | null>(null);
  const [revisions, setRevisions] = useState<DriveRevision[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => {
      api.listFiles(search).then(setFiles).catch(() => {});
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

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
          <div className="max-h-48 overflow-y-auto space-y-1">
            {files
              .filter((f) => !f.is_folder)
              .map((f) => (
                <button
                  key={f.file_id}
                  onClick={() => setSelectedFile(f)}
                  className={`w-full text-left px-3 py-2 rounded text-sm hover:bg-accent ${
                    selectedFile?.file_id === f.file_id ? "bg-accent" : ""
                  }`}
                >
                  {f.name}
                </button>
              ))}
          </div>
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
                        {rev.size ? `${(Number(rev.size) / 1024).toFixed(1)} KB` : "—"}
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
