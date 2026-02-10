import { useCallback } from "react";
import { usePolling } from "./usePolling";
import * as api from "../api/monitor";
import type { MonitorStatus } from "../types/monitor";

export function useMonitorStatus(pollInterval: number = 5000) {
  const { data, error, loading, refresh } = usePolling<MonitorStatus>(
    api.fetchMonitorStatus,
    pollInterval,
  );

  const start = useCallback(async () => {
    await api.startMonitor();
    await refresh();
  }, [refresh]);

  const stop = useCallback(async () => {
    await api.stopMonitor();
    await refresh();
  }, [refresh]);

  return { status: data, error, loading, start, stop, refresh };
}
