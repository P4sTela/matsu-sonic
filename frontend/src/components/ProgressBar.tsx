import { Progress } from "@/components/ui/progress";
import type { SyncProgress } from "@/api/types";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

export function SyncProgressBar({ progress }: { progress: SyncProgress }) {
  const total = progress.total_files || 1;
  const done = progress.completed_files + progress.skipped_files + progress.failed_files;
  const percent = Math.round((done / total) * 100);

  return (
    <div className="space-y-2">
      <div className="flex justify-between text-sm text-muted-foreground">
        <span>
          {done} / {progress.total_files} files
        </span>
        <span>{formatBytes(progress.bytes_downloaded)}</span>
      </div>
      <Progress value={percent} className="h-3" />
      {progress.current_file && (
        <div className="flex items-center gap-2 text-sm">
          <span className="truncate text-muted-foreground">
            {progress.current_file}
          </span>
          <span className="text-xs text-muted-foreground whitespace-nowrap">
            {Math.round(progress.current_file_progress * 100)}%
          </span>
        </div>
      )}
    </div>
  );
}
