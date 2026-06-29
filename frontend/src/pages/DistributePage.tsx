import { useState, useEffect } from "react";
import { Send, Trash2, Plus, Pencil, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FileTreePicker } from "@/components/FileTreePicker";
import * as api from "@/api/client";
import type { DistTarget, DistJob, SyncedFile } from "@/api/types";
import { toast } from "sonner";

export function DistributePage() {
  const [targets, setTargets] = useState<DistTarget[]>([]);
  const [jobs, setJobs] = useState<DistJob[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingName, setEditingName] = useState<string | null>(null);
  const [newTarget, setNewTarget] = useState<Partial<DistTarget>>({ type: "local" });

  // File selection
  const [files, setFiles] = useState<SyncedFile[]>([]);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [selectedTarget, setSelectedTarget] = useState("");
  const [destDir, setDestDir] = useState("");
  const [distributing, setDistributing] = useState(false);

  const load = () => {
    api.listTargets().then((t) => setTargets(t ?? [])).catch(() => {});
    api.listDistJobs().then((j) => setJobs(j ?? [])).catch(() => {});
    api.listFiles().then((f) => setFiles(f ?? [])).catch(() => {});
  };

  useEffect(load, []);

  const openAdd = () => {
    setEditingName(null);
    setNewTarget({ type: "local", select_patterns: [] });
    setDialogOpen(true);
  };

  const openEdit = (t: DistTarget) => {
    setEditingName(t.name);
    setNewTarget({ ...t, select_patterns: t.select_patterns ?? [] });
    setDialogOpen(true);
  };

  const handleSaveTarget = async () => {
    if (!newTarget.name) return;
    if (newTarget.type === "smb") {
      if (!newTarget.server || !newTarget.share) return;
    } else {
      if (!newTarget.path) return;
    }
    // Drop blank patterns before saving.
    const cleaned = {
      ...newTarget,
      select_patterns: (newTarget.select_patterns ?? []).map((p) => p.trim()).filter(Boolean),
    } as DistTarget;
    try {
      if (editingName) {
        await api.updateTarget(editingName, cleaned);
      } else {
        await api.addTarget(cleaned);
      }
      setDialogOpen(false);
      setEditingName(null);
      setNewTarget({ type: "local" });
      load();
    } catch (e) {
      toast.error(editingName ? "Failed to update target" : "Failed to add target", {
        description: e instanceof Error ? e.message : undefined,
      });
    }
  };

  const handleRemove = async (name: string) => {
    await api.removeTarget(name);
    load();
  };

  const toggleFile = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleDistribute = async () => {
    if (selectedIds.size === 0 || !selectedTarget) return;
    setDistributing(true);
    try {
      const results = await api.distribute([...selectedIds], selectedTarget, destDir.trim() || undefined);
      const skipped = results.filter((r) => r.status === "skipped").length;
      const completed = results.filter((r) => r.status === "completed").length;
      const failed = results.filter((r) => r.status === "failed").length;
      if (skipped > 0) {
        toast.warning(`${skipped} file(s) skipped (excluded by target patterns)`, {
          description: `${completed} distributed${failed > 0 ? `, ${failed} failed` : ""}.`,
        });
      } else if (failed > 0) {
        toast.error(`${failed} file(s) failed`, { description: `${completed} distributed.` });
      }
      setSelectedIds(new Set());
      load();
    } catch (e) {
      toast.error("Failed to distribute files", {
        description: e instanceof Error ? e.message : undefined,
      });
    } finally {
      setDistributing(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Targets */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>Distribution Targets</span>
            <Button size="sm" onClick={openAdd}>
              <Plus className="mr-2 h-4 w-4" />
              Add Target
            </Button>
            <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>{editingName ? "Edit Distribution Target" : "Add Distribution Target"}</DialogTitle>
                </DialogHeader>
                <div className="space-y-4">
                  <div>
                    <Label>Name</Label>
                    <Input
                      value={newTarget.name || ""}
                      onChange={(e) => setNewTarget({ ...newTarget, name: e.target.value })}
                      placeholder="e.g. backup-drive"
                      disabled={!!editingName}
                    />
                  </div>
                  <div>
                    <Label>Type</Label>
                    <select
                      value={newTarget.type || "local"}
                      onChange={(e) => {
                        const type = e.target.value as "local" | "smb";
                        if (type === "local") {
                          setNewTarget({ name: newTarget.name, type, path: "" });
                        } else {
                          setNewTarget({ name: newTarget.name, type });
                        }
                      }}
                      className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm"
                    >
                      <option value="local">Local Path</option>
                      <option value="smb">SMB Share</option>
                    </select>
                  </div>
                  {newTarget.type === "smb" ? (
                    <>
                      <div>
                        <Label>Server</Label>
                        <Input
                          value={newTarget.server || ""}
                          onChange={(e) => setNewTarget({ ...newTarget, server: e.target.value })}
                          placeholder="e.g. 192.168.1.10 or PC-NAME"
                        />
                      </div>
                      <div>
                        <Label>Share</Label>
                        <Input
                          value={newTarget.share || ""}
                          onChange={(e) => setNewTarget({ ...newTarget, share: e.target.value })}
                          placeholder="e.g. shared-folder"
                        />
                      </div>
                      <div>
                        <Label>Username</Label>
                        <Input
                          value={newTarget.username || ""}
                          onChange={(e) => setNewTarget({ ...newTarget, username: e.target.value })}
                          placeholder="user"
                        />
                      </div>
                      <div>
                        <Label>Password</Label>
                        <Input
                          type="password"
                          value={newTarget.password || ""}
                          onChange={(e) => setNewTarget({ ...newTarget, password: e.target.value })}
                          placeholder="password"
                        />
                      </div>
                      <div>
                        <Label>Domain (optional)</Label>
                        <Input
                          value={newTarget.domain || ""}
                          onChange={(e) => setNewTarget({ ...newTarget, domain: e.target.value })}
                          placeholder="WORKGROUP"
                        />
                      </div>
                    </>
                  ) : (
                    <div>
                      <Label>Path</Label>
                      <Input
                        value={newTarget.path || ""}
                        onChange={(e) => setNewTarget({ ...newTarget, path: e.target.value })}
                        placeholder="/path/to/destination"
                      />
                    </div>
                  )}
                  <div>
                    <Label>Select Patterns (optional)</Label>
                    <p className="text-xs text-muted-foreground mb-2">
                      Only files matching these patterns are distributed to this target.
                      When empty, all selected files are sent. Matched against the path relative to
                      the sync root (e.g. <code>videos/2024</code> as a prefix, <code>videos/*</code>,
                      {" "}<code>**/*.mp4</code>).
                    </p>
                    <div className="space-y-2">
                      {(newTarget.select_patterns ?? []).map((pattern, i) => (
                        <div key={i} className="flex items-center gap-2">
                          <Input
                            value={pattern}
                            onChange={(e) => {
                              const next = [...(newTarget.select_patterns ?? [])];
                              next[i] = e.target.value;
                              setNewTarget({ ...newTarget, select_patterns: next });
                            }}
                            className="flex-1 font-mono text-sm"
                            placeholder="videos/2024"
                          />
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => {
                              const next = (newTarget.select_patterns ?? []).filter((_, j) => j !== i);
                              setNewTarget({ ...newTarget, select_patterns: next });
                            }}
                          >
                            <X className="h-4 w-4" />
                          </Button>
                        </div>
                      ))}
                    </div>
                    <Button
                      size="sm"
                      variant="outline"
                      className="mt-2"
                      onClick={() =>
                        setNewTarget({
                          ...newTarget,
                          select_patterns: [...(newTarget.select_patterns ?? []), ""],
                        })
                      }
                    >
                      <Plus className="mr-2 h-4 w-4" />
                      Add Pattern
                    </Button>
                  </div>
                  <Button onClick={handleSaveTarget} className="w-full">
                    <Send className="mr-2 h-4 w-4" />
                    {editingName ? "Save Changes" : "Add"}
                  </Button>
                </div>
              </DialogContent>
            </Dialog>
          </CardTitle>
        </CardHeader>
        <CardContent>
          {targets.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">No targets configured</p>
          ) : (
            <div className="space-y-2">
              {targets.map((t) => (
                <div key={t.name} className="flex items-center justify-between rounded border p-3">
                  <div className="min-w-0">
                    <div>
                      <span className="font-medium">{t.name}</span>
                      <span className="ml-2 text-sm text-muted-foreground">
                        {t.type === "smb" ? `\\\\${t.server}\\${t.share}` : t.path}
                      </span>
                    </div>
                    {(t.select_patterns ?? []).length > 0 && (
                      <div className="mt-1 flex flex-wrap gap-1">
                        {(t.select_patterns ?? []).map((p, i) => (
                          <Badge key={i} variant="outline" className="font-mono text-[10px]">{p}</Badge>
                        ))}
                      </div>
                    )}
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <Badge variant="secondary">{t.type}</Badge>
                    <Button size="sm" variant="ghost" onClick={() => openEdit(t)}>
                      <Pencil className="h-4 w-4" />
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => handleRemove(t.name)}>
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Distribute Files */}
      <Card>
        <CardHeader>
          <CardTitle>Distribute Files</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {files.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">No synced files to distribute</p>
          ) : (
            <>
              <div className="rounded border p-2">
                <FileTreePicker
                  files={files}
                  selectedIds={selectedIds}
                  onToggleSelect={toggleFile}
                  showDetails={false}
                  maxHeightClass="max-h-48"
                />
              </div>

              <div className="flex items-end gap-2">
                <div className="flex-1">
                  <Label>Target</Label>
                  <select
                    value={selectedTarget}
                    onChange={(e) => setSelectedTarget(e.target.value)}
                    className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm"
                  >
                    <option value="">Select target...</option>
                    {targets.map((t) => (
                      <option key={t.name} value={t.name}>{t.name}</option>
                    ))}
                  </select>
                </div>
                <div className="flex-1">
                  <Label>Destination Subdirectory (optional)</Label>
                  <Input
                    value={destDir}
                    onChange={(e) => setDestDir(e.target.value)}
                    placeholder="e.g. 2024/videos"
                  />
                </div>
                <Button
                  onClick={handleDistribute}
                  disabled={selectedIds.size === 0 || !selectedTarget || distributing}
                >
                  <Send className="mr-2 h-4 w-4" />
                  Distribute {selectedIds.size > 0 && `(${selectedIds.size})`}
                </Button>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      {/* Recent Jobs */}
      <Card>
        <CardHeader>
          <CardTitle>Recent Jobs</CardTitle>
        </CardHeader>
        <CardContent>
          {jobs.length === 0 ? (
            <p className="text-sm text-muted-foreground text-center py-4">No distribution jobs</p>
          ) : (
            <div className="space-y-2">
              {jobs.map((j) => (
                <div key={j.id} className="flex items-center justify-between rounded border p-3 text-sm">
                  <span className="truncate">{j.source_path}</span>
                  <div className="flex items-center gap-2">
                    <span className="text-muted-foreground">{j.target_type}</span>
                    <Badge
                      variant={
                        j.status === "completed" ? "default" : j.status === "failed" ? "destructive" : "secondary"
                      }
                    >
                      {j.status}
                    </Badge>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
