import type { Keyword } from "../../types/keyword";
import { KeywordForm } from "./KeywordForm";
import { KeywordBadge } from "./KeywordBadge";

interface KeywordPanelProps {
  keywords: Keyword[];
  loading: boolean;
  onAdd: (value: string) => Promise<void>;
  onRemove: (id: number) => Promise<void>;
  onFilter: (keywordId: number | undefined) => void;
  activeFilter: number | undefined;
}

export function KeywordPanel({
  keywords,
  loading,
  onAdd,
  onRemove,
  onFilter,
  activeFilter,
}: KeywordPanelProps) {
  return (
    <aside className="w-72 shrink-0 flex flex-col gap-4 rounded-lg bg-gray-900 border border-gray-800 p-4">
      <h2 className="text-sm font-semibold uppercase tracking-wider text-gray-400">
        Monitored Keywords
      </h2>

      <KeywordForm onSubmit={onAdd} />

      {loading ? (
        <p className="text-sm text-gray-500">Loading...</p>
      ) : keywords.length === 0 ? (
        <p className="text-sm text-gray-500">
          No keywords configured. Add one above to start monitoring.
        </p>
      ) : (
        <div className="flex flex-col gap-2 overflow-y-auto">
          {/* "All" filter */}
          <button
            onClick={() => onFilter(undefined)}
            className={`text-left text-sm px-2 py-1 rounded ${
              activeFilter === undefined
                ? "bg-gray-700 text-white"
                : "text-gray-400 hover:text-gray-200"
            }`}
          >
            All keywords
          </button>
          {keywords.map((kw) => (
            <KeywordBadge
              key={kw.id}
              keyword={kw}
              onRemove={() => onRemove(kw.id)}
              onFilter={() => onFilter(kw.id)}
              isActive={activeFilter === kw.id}
            />
          ))}
        </div>
      )}
    </aside>
  );
}
