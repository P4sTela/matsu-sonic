import { useState, useEffect } from "react";
import { Search } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { FileTable } from "@/components/FileTable";
import * as api from "@/api/client";
import type { SyncedFile } from "@/api/types";

export function FilesPage() {
  const [files, setFiles] = useState<SyncedFile[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const timer = setTimeout(() => {
      setLoading(true);
      api
        .listFiles(search)
        .then((f) => setFiles(f ?? []))
        .catch(() => setFiles([]))
        .finally(() => setLoading(false));
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Synced Files</CardTitle>
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
          <FileTable files={files} />
        )}
      </CardContent>
    </Card>
  );
}
