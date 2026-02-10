interface MetricCardProps {
  label: string;
  value: number;
  suffix?: string;
}

export function MetricCard({ label, value, suffix }: MetricCardProps) {
  return (
    <div className="flex flex-col">
      <span className="text-xs text-gray-500 uppercase tracking-wider">
        {label}
      </span>
      <span className="text-lg font-mono font-semibold tabular-nums">
        {value.toLocaleString()}
        {suffix && <span className="text-xs text-gray-500 ml-1">{suffix}</span>}
      </span>
    </div>
  );
}
