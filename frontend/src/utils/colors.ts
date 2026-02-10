interface KeywordColors {
  dot: string; // Small indicator dot
  badge: string; // Inline badge (keyword name)
  border: string; // Panel border accent
  activeBg: string; // Active filter background
  rowHighlight: string; // Table row left-border highlight
}

const PALETTE: KeywordColors[] = [
  {
    dot: "bg-red-400",
    badge: "bg-red-500/20 text-red-300",
    border: "border-red-500/30",
    activeBg: "bg-red-500/10",
    rowHighlight: "border-l-2 border-l-red-500",
  },
  {
    dot: "bg-amber-400",
    badge: "bg-amber-500/20 text-amber-300",
    border: "border-amber-500/30",
    activeBg: "bg-amber-500/10",
    rowHighlight: "border-l-2 border-l-amber-500",
  },
  {
    dot: "bg-emerald-400",
    badge: "bg-emerald-500/20 text-emerald-300",
    border: "border-emerald-500/30",
    activeBg: "bg-emerald-500/10",
    rowHighlight: "border-l-2 border-l-emerald-500",
  },
  {
    dot: "bg-sky-400",
    badge: "bg-sky-500/20 text-sky-300",
    border: "border-sky-500/30",
    activeBg: "bg-sky-500/10",
    rowHighlight: "border-l-2 border-l-sky-500",
  },
  {
    dot: "bg-violet-400",
    badge: "bg-violet-500/20 text-violet-300",
    border: "border-violet-500/30",
    activeBg: "bg-violet-500/10",
    rowHighlight: "border-l-2 border-l-violet-500",
  },
  {
    dot: "bg-fuchsia-400",
    badge: "bg-fuchsia-500/20 text-fuchsia-300",
    border: "border-fuchsia-500/30",
    activeBg: "bg-fuchsia-500/10",
    rowHighlight: "border-l-2 border-l-fuchsia-500",
  },
  {
    dot: "bg-cyan-400",
    badge: "bg-cyan-500/20 text-cyan-300",
    border: "border-cyan-500/30",
    activeBg: "bg-cyan-500/10",
    rowHighlight: "border-l-2 border-l-cyan-500",
  },
  {
    dot: "bg-rose-400",
    badge: "bg-rose-500/20 text-rose-300",
    border: "border-rose-500/30",
    activeBg: "bg-rose-500/10",
    rowHighlight: "border-l-2 border-l-rose-500",
  },
];

export function getKeywordColor(keywordId: number): KeywordColors {
  return PALETTE[keywordId % PALETTE.length];
}
