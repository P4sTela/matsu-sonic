import { useState, useEffect } from "react";
import { Play, RefreshCw, Square, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { SyncProgressBar } from "@/components/ProgressBar";
import { useSync } from "@/hooks/useSync";
import * as api from "@/api/client";
import type { SyncRun } from "@/api/types";

export function SyncPage() {
  const { progress, startFull, startIncremental, cancel } = useSync();
  const [history, setHistory] = useState<SyncRun[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.getSyncHistory().then((runs) => setHistory(runs ?? [])).catch(() => {});
  }, [progress.is_running]);

  const handleSync = async (mode: "full" | "incremental") => {
    try {
      setError(null);
      if (mode === "full") await startFull();
      else await startIncremental();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to start sync");
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>Sync</span>
            {progress.is_running && (
              <Badge variant="default">Running</Badge>
            )}
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
            {progress.is_running && (
              <Button variant="destructive" onClick={cancel}>
                <Square className="mr-2 h-4 w-4" />
                Cancel
              </Button>
            )}
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
                <div key={i} className="text-sm text-destructive">{err}</div>
              ))}
            </ScrollArea>
          )}
        </CardContent>
      </Card>

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
