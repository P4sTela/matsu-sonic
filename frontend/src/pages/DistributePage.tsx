import { useState, useEffect } from "react";
import { Send, Trash2, Plus } from "lucide-react";
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
  DialogTrigger,
} from "@/components/ui/dialog";
import * as api from "@/api/client";
import type { DistTarget, DistJob } from "@/api/types";

export function DistributePage() {
  const [targets, setTargets] = useState<DistTarget[]>([]);
  const [jobs, setJobs] = useState<DistJob[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [newTarget, setNewTarget] = useState<Partial<DistTarget>>({ type: "local" });

  const load = () => {
    api.listTargets().then(setTargets).catch(() => {});
    api.listDistJobs().then(setJobs).catch(() => {});
  };

  useEffect(load, []);

  const handleAdd = async () => {
    if (!newTarget.name || !newTarget.path) return;
    try {
      await api.addTarget(newTarget as DistTarget);
      setDialogOpen(false);
      setNewTarget({ type: "local" });
      load();
    } catch {
      // TODO: show error
    }
  };

  const handleRemove = async (name: string) => {
    await api.removeTarget(name);
    load();
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>Distribution Targets</span>
            <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
              <DialogTrigger render={<Button size="sm" />}>
                <Plus className="mr-2 h-4 w-4" />
                Add Target
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Add Distribution Target</DialogTitle>
                </DialogHeader>
                <div className="space-y-4">
                  <div>
                    <Label>Name</Label>
                    <Input
                      value={newTarget.name || ""}
                      onChange={(e) => setNewTarget({ ...newTarget, name: e.target.value })}
                      placeholder="e.g. backup-drive"
                    />
                  </div>
                  <div>
                    <Label>Path</Label>
                    <Input
                      value={newTarget.path || ""}
                      onChange={(e) => setNewTarget({ ...newTarget, path: e.target.value })}
                      placeholder="/path/to/destination"
                    />
                  </div>
                  <Button onClick={handleAdd} className="w-full">
                    <Send className="mr-2 h-4 w-4" />
                    Add
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
                  <div>
                    <span className="font-medium">{t.name}</span>
                    <span className="ml-2 text-sm text-muted-foreground">{t.path}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">{t.type}</Badge>
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
