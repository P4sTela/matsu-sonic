import { useState, useEffect, useCallback } from "react";
import type { Config } from "@/api/types";
import * as api from "@/api/client";

export function useConfig() {
  const [config, setConfig] = useState<Config | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      setLoading(true);
      const cfg = await api.getConfig();
      setConfig(cfg);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load config");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const save = useCallback(async (update: Partial<Config>) => {
    try {
      await api.updateConfig(update);
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save config");
    }
  }, [load]);

  return { config, loading, error, save, reload: load };
}
