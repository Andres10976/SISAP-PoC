import type { Keyword } from "../../types/keyword";
import { getKeywordColor } from "../../utils/colors";

interface KeywordBadgeProps {
  keyword: Keyword;
  onRemove: () => void;
  onFilter: () => void;
  isActive: boolean;
}

export function KeywordBadge({
  keyword,
  onRemove,
  onFilter,
  isActive,
}: KeywordBadgeProps) {
  const color = getKeywordColor(keyword.id);

  return (
    <div
      className={`flex items-center justify-between rounded-md px-2 py-1.5
                  border transition-colors cursor-pointer ${color.border} ${
                    isActive ? color.activeBg : "hover:bg-gray-800"
                  }`}
      onClick={onFilter}
    >
      <div className="flex items-center gap-2">
        <span className={`h-2.5 w-2.5 rounded-full ${color.dot}`} />
        <span className="text-sm font-medium">{keyword.value}</span>
      </div>
      <button
        onClick={(e) => {
          e.stopPropagation();
          onRemove();
        }}
        className="text-gray-500 hover:text-red-400 text-xs ml-2"
        aria-label={`Remove ${keyword.value}`}
      >
        ✕
      </button>
    </div>
  );
}
