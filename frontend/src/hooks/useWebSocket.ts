import { useEffect, useRef } from "react";
import { SyncWebSocket } from "@/api/websocket";
import type { WSMessage } from "@/api/types";

export function useWebSocket(onMessage: (msg: WSMessage) => void) {
  const wsRef = useRef<SyncWebSocket | null>(null);

  useEffect(() => {
    const ws = new SyncWebSocket();
    wsRef.current = ws;

    const unsub = ws.subscribe(onMessage);
    ws.connect();

    return () => {
      unsub();
      ws.disconnect();
    };
  }, [onMessage]);

  return wsRef;
}
