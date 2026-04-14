import { useState, useEffect, useCallback } from "react";
import { Folder, File, ArrowUp, Check, FolderPlus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import * as api from "@/api/client";
import type { BrowseResult } from "@/api/types";

/** If absPath is under cwd, return the relative path; otherwise return absPath as-is. */
function toRelative(absPath: string, cwd: string): string {
  const cwdSlash = cwd.endsWith("/") ? cwd : cwd + "/";
  if (absPath === cwd) return ".";
  if (absPath.startsWith(cwdSlash)) return absPath.slice(cwdSlash.length);
  return absPath;
}

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelect: (path: string) => void;
  title?: string;
  /** "file" allows selecting files, "directory" only directories */
  mode?: "file" | "directory";
}

export function DirBrowser({ open, onOpenChange, onSelect, title = "Select Path", mode = "directory" }: Props) {
  const [result, setResult] = useState<BrowseResult | null>(null);
  const [pathInput, setPathInput] = useState("");
  const [loading, setLoading] = useState(false);
  const [newFolderName, setNewFolderName] = useState("");
  const [showNewFolder, setShowNewFolder] = useState(false);

  const browse = useCallback(async (path?: string) => {
    setLoading(true);
    try {
      const res = await api.browseDirectory(path);
      setResult(res);
      setPathInput(res.current);
    } catch {
      // keep current state
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open) browse();
  }, [open, browse]);

  const handleNavigate = (path: string) => {
    browse(path);
  };

  const handleGoTo = () => {
    if (pathInput) browse(pathInput);
  };

  const handleSelect = (absPath: string) => {
    const path = result?.cwd ? toRelative(absPath, result.cwd) : absPath;
    onSelect(path);
    onOpenChange(false);
  };

  const handleCreateFolder = async () => {
    if (!newFolderName.trim() || !result?.current) return;
    const newPath = result.current + "/" + newFolderName.trim();
    try {
      await api.makeDirectory(newPath);
      setNewFolderName("");
      setShowNewFolder(false);
      browse(result.current);
    } catch {
      // TODO: show error
    }
  };

  const isUnderCwd = result?.cwd && result.current.startsWith(result.cwd);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div className="flex gap-2">
            <Input
              value={pathInput}
              onChange={(e) => setPathInput(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && handleGoTo()}
              className="text-xs font-mono"
            />
            {result?.parent && (
              <Button size="sm" variant="outline" onClick={() => handleNavigate(result.parent)}>
                <ArrowUp className="h-4 w-4" />
              </Button>
            )}
          </div>

          {isUnderCwd && (
            <div className="text-xs text-muted-foreground flex items-center gap-1">
              <Badge variant="secondary" className="text-[10px]">relative</Badge>
              {toRelative(result.current, result.cwd)}
            </div>
          )}

          <ScrollArea className="h-64 rounded border">
            {loading ? (
              <p className="text-sm text-muted-foreground text-center py-8">Loading...</p>
            ) : (
              <div className="p-1">
                {(result?.items ?? []).map((item) => (
                  <div
                    key={item.path}
                    className="flex items-center justify-between rounded px-2 py-1.5 hover:bg-accent text-sm"
                  >
                    <button
                      className="flex items-center gap-2 flex-1 text-left min-w-0"
                      onClick={() => item.is_dir ? handleNavigate(item.path) : undefined}
                    >
                      {item.is_dir ? (
                        <Folder className="h-4 w-4 text-muted-foreground shrink-0" />
                      ) : (
                        <File className="h-4 w-4 text-muted-foreground shrink-0" />
                      )}
                      <span className="truncate">{item.name}</span>
                    </button>
                    {((mode === "file" && !item.is_dir) || (mode === "directory" && item.is_dir)) && (
                      <Button
                        size="sm"
                        variant="ghost"
                        className="shrink-0 h-7 w-7 p-0"
                        onClick={() => handleSelect(item.path)}
                      >
                        <Check className="h-3.5 w-3.5" />
                      </Button>
                    )}
                  </div>
                ))}
                {result?.items?.length === 0 && (
                  <p className="text-sm text-muted-foreground text-center py-4">Empty directory</p>
                )}
              </div>
            )}
          </ScrollArea>

          {mode === "directory" && (
            <div className="space-y-2">
              {showNewFolder ? (
                <div className="flex gap-2">
                  <Input
                    value={newFolderName}
                    onChange={(e) => setNewFolderName(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && handleCreateFolder()}
                    placeholder="New folder name"
                    autoFocus
                    className="flex-1"
                  />
                  <Button size="sm" onClick={handleCreateFolder} disabled={!newFolderName.trim()}>
                    Create
                  </Button>
                  <Button size="sm" variant="ghost" onClick={() => { setShowNewFolder(false); setNewFolderName(""); }}>
                    Cancel
                  </Button>
                </div>
              ) : (
                <Button variant="outline" size="sm" onClick={() => setShowNewFolder(true)}>
                  <FolderPlus className="mr-2 h-4 w-4" />
                  New Folder
                </Button>
              )}

              {result?.current && (
                <Button className="w-full" onClick={() => handleSelect(result.current)}>
                  <Check className="mr-2 h-4 w-4" />
                  Select this folder
                  {isUnderCwd && (
                    <span className="ml-1 text-xs opacity-70">
                      ({toRelative(result.current, result.cwd)})
                    </span>
                  )}
                </Button>
              )}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
