export function Header() {
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
    </header>
  );
}
