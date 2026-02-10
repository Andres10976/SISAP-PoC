export function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center h-48 text-gray-500 gap-2">
      <p className="text-sm">No matched certificates yet.</p>
      <p className="text-xs">
        Add keywords and start the monitor to begin scanning CT logs.
      </p>
    </div>
  );
}
