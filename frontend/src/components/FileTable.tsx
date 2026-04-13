import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import type { SyncedFile } from "@/api/types";

function formatBytes(bytes: number): string {
  if (bytes === 0) return "—";
  const units = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

interface Props {
  files: SyncedFile[];
  onSelect?: (file: SyncedFile) => void;
}

export function FileTable({ files, onSelect }: Props) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
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
            className={onSelect ? "cursor-pointer" : ""}
            onClick={() => onSelect?.(f)}
          >
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
            <TableCell colSpan={4} className="text-center text-muted-foreground py-8">
              No files synced yet
            </TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
  );
}
