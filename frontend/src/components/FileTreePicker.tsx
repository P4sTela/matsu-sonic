import { useState, useMemo } from "react";
import { Check, ChevronRight, ChevronDown, Folder, File } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { SyncedFile } from "@/api/types";
import { formatBytes } from "@/lib/utils";

interface TreeNode {
  file: SyncedFile;
  children: TreeNode[];
}

function buildTree(files: SyncedFile[]): TreeNode[] {
  const byId = new Map<string, SyncedFile>();
  const childrenMap = new Map<string, TreeNode[]>();

  for (const f of files) {
    byId.set(f.file_id, f);
  }

  for (const f of files) {
    const pid = f.parent_id || "__root__";
    if (!childrenMap.has(pid)) childrenMap.set(pid, []);
    childrenMap.get(pid)!.push({ file: f, children: [] });
  }

  function attach(nodes: TreeNode[]): TreeNode[] {
    for (const node of nodes) {
      const kids = childrenMap.get(node.file.file_id);
      if (kids) {
        node.children = attach(kids);
      }
    }
    return nodes.sort((a, b) => {
      if (a.file.is_folder !== b.file.is_folder) return a.file.is_folder ? -1 : 1;
      return a.file.name.localeCompare(b.file.name);
    });
  }

  const roots: TreeNode[] = [];
  for (const f of files) {
    if (!f.parent_id || !byId.has(f.parent_id)) {
      const node = childrenMap.get(f.parent_id || "__root__")?.find(n => n.file.file_id === f.file_id);
      if (node) roots.push(node);
    }
  }

  return attach(roots);
}

function flattenTree(nodes: TreeNode[], expanded: Set<string>, depth = 0): { file: SyncedFile; depth: number }[] {
  const result: { file: SyncedFile; depth: number }[] = [];
  for (const node of nodes) {
    result.push({ file: node.file, depth });
    if (node.file.is_folder && expanded.has(node.file.file_id)) {
      result.push(...flattenTree(node.children, expanded, depth + 1));
    }
  }
  return result;
}

interface Props {
  files: SyncedFile[];
  selectedIds: Set<string>;
  onToggleSelect: (id: string) => void;
  /** Show size and last synced columns (default: true) */
  showDetails?: boolean;
  /** Show checkboxes for file selection (default: true) */
  showCheckbox?: boolean;
  /** Max height with scroll (e.g. "max-h-64") */
  maxHeightClass?: string;
}

export function FileTreePicker({
  files,
  selectedIds,
  onToggleSelect,
  showDetails = true,
  showCheckbox = true,
  maxHeightClass,
}: Props) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const tree = useMemo(() => buildTree(files), [files]);
  const rows = useMemo(() => flattenTree(tree, expanded), [tree, expanded]);
  const folderCount = useMemo(() => files.filter(f => f.is_folder).length, [files]);

  const toggleExpand = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const expandAll = () => {
    setExpanded(new Set(files.filter(f => f.is_folder).map(f => f.file_id)));
  };

  const collapseAll = () => {
    setExpanded(new Set());
  };

  return (
    <div className="space-y-2">
      {folderCount > 0 && (
        <div className="flex gap-1 justify-end">
          <Button size="sm" variant="outline" onClick={expandAll}>Expand</Button>
          <Button size="sm" variant="outline" onClick={collapseAll}>Collapse</Button>
        </div>
      )}
      <div className={`overflow-y-auto ${maxHeightClass ?? ""}`}>
        {rows.map(({ file: f, depth }) => (
          <div
            key={f.file_id}
            className={`flex items-center gap-2 px-2 py-1.5 rounded text-sm ${
              f.is_folder
                ? "cursor-pointer hover:bg-accent/50"
                : `cursor-pointer hover:bg-accent ${selectedIds.has(f.file_id) ? "bg-accent" : ""}`
            }`}
            onClick={() => f.is_folder ? toggleExpand(f.file_id) : onToggleSelect(f.file_id)}
          >
            {/* Checkbox (files only) */}
            {showCheckbox && (
              <div className="w-4 shrink-0">
                {!f.is_folder && (
                  <div className={`h-4 w-4 rounded border flex items-center justify-center ${
                    selectedIds.has(f.file_id) ? "bg-primary border-primary" : "border-input"
                  }`}>
                    {selectedIds.has(f.file_id) && <Check className="h-3 w-3 text-primary-foreground" />}
                  </div>
                )}
              </div>
            )}

            {/* Name with indent */}
            <div className="flex items-center gap-1.5 flex-1 min-w-0" style={{ paddingLeft: `${depth * 20}px` }}>
              {f.is_folder ? (
                <>
                  {expanded.has(f.file_id) ? (
                    <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" />
                  ) : (
                    <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />
                  )}
                  <Folder className="h-4 w-4 text-muted-foreground shrink-0" />
                  <span className="font-medium truncate">{f.name}</span>
                </>
              ) : (
                <>
                  <span className="w-4 shrink-0" />
                  <File className="h-4 w-4 text-muted-foreground shrink-0" />
                  <span className="truncate">{f.name}</span>
                </>
              )}
            </div>

            {/* Details */}
            {showDetails && !f.is_folder && (
              <>
                <span className="text-muted-foreground text-xs shrink-0 w-16 text-right">
                  {formatBytes(f.size)}
                </span>
                <span className="text-muted-foreground text-xs shrink-0 w-32 text-right hidden sm:block">
                  {f.last_synced ? new Date(f.last_synced).toLocaleString() : "—"}
                </span>
              </>
            )}
          </div>
        ))}
        {rows.length === 0 && (
          <p className="text-sm text-muted-foreground text-center py-4">No files</p>
        )}
      </div>
    </div>
  );
}
