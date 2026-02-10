import type { MonitorStatus } from "../../types/monitor";
import { MetricCard } from "./MetricCard";
import { ExportButton } from "../export/ExportButton";

interface StatusBarProps {
  status: MonitorStatus | null;
  loading: boolean;
  onStart: () => void;
  onStop: () => void;
}

export function StatusBar({
  status,
  loading,
  onStart,
  onStop,
}: StatusBarProps) {
  const isRunning = status?.is_running ?? false;

  return (
    <div className="flex items-center gap-4 rounded-lg bg-gray-900 border border-gray-800 p-4">
      {/* Start / Stop button */}
      <button
        onClick={isRunning ? onStop : onStart}
        disabled={loading}
        className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
          isRunning
            ? "bg-red-600 hover:bg-red-700 text-white"
            : "bg-emerald-600 hover:bg-emerald-700 text-white"
        } disabled:opacity-50`}
      >
        {isRunning ? "Stop Monitor" : "Start Monitor"}
      </button>

      {/* Metrics */}
      <div className="flex gap-4 flex-1">
        <MetricCard
          label="Total Processed"
          value={status?.total_processed ?? 0}
        />
        <MetricCard
          label="Last Batch"
          value={status?.certs_in_last_cycle ?? 0}
          suffix="certs"
        />
        <MetricCard
          label="Last Matches"
          value={status?.matches_in_last_cycle ?? 0}
        />
        <MetricCard label="Tree Size" value={status?.last_tree_size ?? 0} />
      </div>

      {/* Error / last run time */}
      {status?.last_error ? (
        <span className="text-xs text-red-400 max-w-xs truncate" title={status.last_error}>
          Error: {status.last_error}
        </span>
      ) : status?.last_run_at ? (
        <span className="text-xs text-gray-500">
          Last run: {new Date(status.last_run_at).toLocaleTimeString()}
        </span>
      ) : null}

      <ExportButton />
    </div>
  );
}
