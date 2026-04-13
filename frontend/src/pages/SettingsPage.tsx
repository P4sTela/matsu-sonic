import { useState } from "react";
import { Save, CheckCircle, AlertCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { useConfig } from "@/hooks/useConfig";
import * as api from "@/api/client";

export function SettingsPage() {
  const { config, loading, error, save } = useConfig();
  const [authStatus, setAuthStatus] = useState<string | null>(null);
  const [authUser, setAuthUser] = useState<string | null>(null);

  const handleTestAuth = async () => {
    try {
      setAuthStatus("testing");
      const result = await api.testAuth();
      setAuthStatus("ok");
      setAuthUser(`${result.user.displayName} (${result.user.emailAddress})`);
    } catch (e) {
      setAuthStatus("error");
      setAuthUser(e instanceof Error ? e.message : "Auth failed");
    }
  };

  if (loading) return <p className="text-center text-muted-foreground py-8">Loading...</p>;
  if (!config) return <p className="text-center text-destructive py-8">{error}</p>;

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle>Authentication</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <Label>Auth Method</Label>
            <Input value={config.auth_method} disabled />
          </div>
          <div>
            <Label>Credentials Path</Label>
            <Input
              value={config.credentials_path}
              onChange={(e) => save({ credentials_path: e.target.value })}
            />
          </div>
          <div className="flex items-center gap-2">
            <Button variant="secondary" onClick={handleTestAuth}>
              Test Auth
            </Button>
            {authStatus === "ok" && (
              <span className="flex items-center gap-1 text-sm text-green-600">
                <CheckCircle className="h-4 w-4" />
                {authUser}
              </span>
            )}
            {authStatus === "error" && (
              <span className="flex items-center gap-1 text-sm text-destructive">
                <AlertCircle className="h-4 w-4" />
                {authUser}
              </span>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Sync Settings</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <Label>Sync Folder ID</Label>
            <Input
              value={config.sync_folder_id}
              onChange={(e) => save({ sync_folder_id: e.target.value })}
              placeholder="Google Drive Folder ID"
            />
          </div>
          <div>
            <Label>Local Sync Directory</Label>
            <Input
              value={config.local_sync_dir}
              onChange={(e) => save({ local_sync_dir: e.target.value })}
              placeholder="/path/to/sync"
            />
          </div>
          <Separator />
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>Max Workers</Label>
              <Input
                type="number"
                value={config.max_workers}
                onChange={(e) => save({ max_workers: Number(e.target.value) })}
                min={1}
                max={10}
              />
            </div>
            <div>
              <Label>Chunk Size (MB)</Label>
              <Input
                type="number"
                value={config.chunk_size_mb}
                onChange={(e) => save({ chunk_size_mb: Number(e.target.value) })}
                min={1}
                max={100}
              />
            </div>
          </div>
          <Button onClick={() => save(config)}>
            <Save className="mr-2 h-4 w-4" />
            Save All
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}
