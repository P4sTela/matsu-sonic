import { useState, useEffect, useCallback, useMemo } from "react";
import { Folder, File, ArrowUp, Check, EyeOff, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import * as api from "@/api/client";
import type { DriveBrowseResult } from "@/api/types";

type Source = "my_drive" | "shared";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSelect: (folderId: string, folderName: string) => void;
  onIgnore?: (name: string) => void;
  title?: string;
}

export function DriveBrowser({ open, onOpenChange, onSelect, onIgnore, title = "Select Drive Folder" }: Props) {
  const [result, setResult] = useState<DriveBrowseResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [path, setPath] = useState<{ id: string; name: string }[]>([]);
  const [source, setSource] = useState<Source>("my_drive");
  const [filter, setFilter] = useState("");

  const browse = useCallback(async (folderId?: string, src?: Source) => {
    setLoading(true);
    setError(null);
    try {
      const res = await api.browseDrive(folderId, src);
      setResult(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to browse Drive");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open) {
      setPath([]);
      setSource("my_drive");
      setFilter("");
      browse(undefined, "my_drive");
    }
  }, [open, browse]);

  const switchSource = (src: Source) => {
    setSource(src);
    setPath([]);
    browse(undefined, src);
  };

  const handleNavigate = (id: string, name: string) => {
    setPath((prev) => [...prev, { id, name }]);
    browse(id, source);
  };

  const handleUp = () => {
    if (!result?.parent_id) return;
    setPath((prev) => prev.slice(0, -1));
    browse(result.parent_id || undefined, source);
  };

  const handleBreadcrumb = (index: number) => {
    if (index < 0) {
      setPath([]);
      browse(undefined, source);
    } else {
      const target = path[index];
      setPath((prev) => prev.slice(0, index + 1));
      browse(target.id, source);
    }
  };

  const handleSelect = () => {
    if (!result) return;
    onSelect(result.folder_id, result.folder_name);
    onOpenChange(false);
  };

  const filteredItems = useMemo(() => {
    const items = result?.items ?? [];
    if (!filter) return items;
    const lower = filter.toLowerCase();
    return items.filter((item) => item.name.toLowerCase().includes(lower));
  }, [result, filter]);

  const rootLabel = source === "shared" ? "Shared with me" : "My Drive";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          {/* Source tabs */}
          <div className="flex gap-1 rounded-md border p-1">
            <button
              onClick={() => switchSource("my_drive")}
              className={`flex-1 rounded px-3 py-1.5 text-sm transition-colors ${
                source === "my_drive"
                  ? "bg-accent text-accent-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              My Drive
            </button>
            <button
              onClick={() => switchSource("shared")}
              className={`flex-1 rounded px-3 py-1.5 text-sm transition-colors ${
                source === "shared"
                  ? "bg-accent text-accent-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground"
              }`}
            >
              Shared with me
            </button>
          </div>

          {/* Breadcrumb */}
          <div className="flex items-center gap-1 text-sm overflow-x-auto">
            {result?.parent_id && (
              <Button size="sm" variant="outline" onClick={handleUp} className="shrink-0">
                <ArrowUp className="h-4 w-4" />
              </Button>
            )}
            <button
              onClick={() => handleBreadcrumb(-1)}
              className="text-muted-foreground hover:text-foreground shrink-0"
            >
              {rootLabel}
            </button>
            {path.map((p, i) => (
              <span key={p.id} className="flex items-center gap-1 shrink-0">
                <span className="text-muted-foreground">/</span>
                <button
                  onClick={() => handleBreadcrumb(i)}
                  className="text-muted-foreground hover:text-foreground truncate max-w-32"
                >
                  {p.name}
                </button>
              </span>
            ))}
          </div>

          {error && (
            <p className="text-sm text-destructive">{error}</p>
          )}

          <div className="relative">
            <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Filter..."
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="pl-9"
            />
          </div>

          <ScrollArea className="h-96 rounded border">
            {loading ? (
              <p className="text-sm text-muted-foreground text-center py-8">Loading...</p>
            ) : (
              <div className="p-1">
                {filteredItems.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-center justify-between rounded px-2 py-1.5 hover:bg-accent text-sm overflow-hidden"
                  >
                    <button
                      className="flex items-center gap-2 flex-1 text-left min-w-0"
                      onClick={() => item.is_folder ? handleNavigate(item.id, item.name) : undefined}
                      disabled={!item.is_folder}
                    >
                      {item.is_folder ? (
                        <Folder className="h-4 w-4 text-muted-foreground shrink-0" />
                      ) : (
                        <File className="h-4 w-4 text-muted-foreground shrink-0" />
                      )}
                      <span className="truncate">{item.name}</span>
                    </button>
                    <div className="flex items-center gap-0.5 shrink-0">
                      {onIgnore && (
                        <Button
                          size="sm"
                          variant="ghost"
                          className="h-7 w-7 p-0 text-muted-foreground hover:text-destructive"
                          title={`Ignore "${item.name}"`}
                          onClick={() => onIgnore(item.name)}
                        >
                          <EyeOff className="h-3.5 w-3.5" />
                        </Button>
                      )}
                      {item.is_folder && (
                        <Button
                          size="sm"
                          variant="ghost"
                          className="h-7 w-7 p-0"
                          onClick={() => {
                            onSelect(item.id, item.name);
                            onOpenChange(false);
                          }}
                        >
                          <Check className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </div>
                  </div>
                ))}
                {filteredItems.length === 0 && (
                  <p className="text-sm text-muted-foreground text-center py-4">
                    {filter ? "No matches" : "Empty folder"}
                  </p>
                )}
              </div>
            )}
          </ScrollArea>

          {result && result.folder_id !== "" && (
            <Button className="w-full" onClick={handleSelect}>
              <Check className="mr-2 h-4 w-4" />
              Select &ldquo;{result.folder_name}&rdquo;
            </Button>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
