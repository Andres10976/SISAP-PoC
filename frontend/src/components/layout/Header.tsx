interface HeaderProps {
  isMonitorRunning?: boolean;
}

export function Header({ isMonitorRunning }: HeaderProps) {
  return (
    <header className="h-16 border-b border-gray-800 bg-gray-900 px-6 flex items-center justify-between">
      <div className="flex items-center gap-3">
        <h1 className="text-lg font-semibold tracking-tight">
          CT Brand Monitor
        </h1>
        <span className="text-xs text-gray-500 font-mono">
          Certificate Transparency
        </span>
      </div>
      {isMonitorRunning !== undefined && (
        <div className="flex items-center gap-2 text-sm">
          <span
            className={`h-2 w-2 rounded-full ${
              isMonitorRunning ? "bg-emerald-400 animate-pulse" : "bg-gray-600"
            }`}
          />
          <span className="text-gray-400">
            {isMonitorRunning ? "Monitoring" : "Stopped"}
          </span>
        </div>
      )}
    </header>
  );
}
