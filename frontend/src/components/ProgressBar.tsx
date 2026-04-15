import { Progress } from "@/components/ui/progress";
import type { SyncProgress } from "@/api/types";
import { formatBytes } from "@/lib/utils";

export function SyncProgressBar({ progress }: { progress: SyncProgress }) {
  const total = progress.total_files || 1;
  const done = progress.completed_files + progress.skipped_files + progress.failed_files;
  const percent = Math.round((done / total) * 100);
  const active = progress.active_downloads ?? [];

  return (
    <div className="space-y-3">
      {/* Overall progress */}
      <div className="space-y-1">
        <div className="flex justify-between text-sm text-muted-foreground">
          <span>
            {done} / {progress.total_files} files
          </span>
          <span>{formatBytes(progress.bytes_downloaded)}</span>
        </div>
        <Progress value={percent} className="h-3" />
      </div>

      {/* Per-worker progress */}
      {active.length > 0 && (
        <div className="grid gap-2" style={{ gridTemplateColumns: `repeat(${Math.min(active.length, 3)}, 1fr)` }}>
          {active.map((dl) => (
            <div key={dl.file_id} className="rounded border p-2 space-y-1">
              <div className="text-xs truncate text-muted-foreground" title={dl.file_name}>
                {dl.file_name}
              </div>
              <Progress value={Math.round(dl.progress * 100)} className="h-1.5" />
              <div className="text-[10px] text-muted-foreground text-right">
                {Math.round(dl.progress * 100)}%
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
