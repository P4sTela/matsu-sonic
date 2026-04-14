import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useSyncContext } from "@/hooks/SyncProvider";

export function LogPage() {
  const { logs } = useSyncContext();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Logs</CardTitle>
      </CardHeader>
      <CardContent>
        {logs.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center py-8">
            No logs yet. Start a sync to see activity.
          </p>
        ) : (
          <ScrollArea className="h-[600px]">
            <div className="space-y-1 font-mono text-xs">
              {logs.map((log, i) => (
                <div key={i} className="flex items-start gap-2 py-1 border-b border-border/50">
                  <Badge
                    variant={
                      log.level === "error"
                        ? "destructive"
                        : log.level === "warn"
                          ? "secondary"
                          : "default"
                    }
                    className="text-[10px] shrink-0"
                  >
                    {log.level}
                  </Badge>
                  <span className="text-muted-foreground shrink-0">
                    {new Date(log.ts).toLocaleTimeString()}
                  </span>
                  <span className="break-all">{log.msg}</span>
                </div>
              ))}
            </div>
          </ScrollArea>
        )}
      </CardContent>
    </Card>
  );
}
