import { request } from "./client";
import type { MonitorStatus } from "../types/monitor";

export function fetchMonitorStatus(): Promise<MonitorStatus> {
  return request<MonitorStatus>("/monitor/status");
}

export function startMonitor(): Promise<{ message: string }> {
  return request<{ message: string }>("/monitor/start", { method: "POST" });
}

export function stopMonitor(): Promise<{ message: string }> {
  return request<{ message: string }>("/monitor/stop", { method: "POST" });
}
