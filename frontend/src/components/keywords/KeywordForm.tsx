import { useState, type FormEvent } from "react";

interface KeywordFormProps {
  onSubmit: (value: string) => Promise<void>;
}

export function KeywordForm({ onSubmit }: KeywordFormProps) {
  const [value, setValue] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = value.trim();
    if (!trimmed) return;

    setSubmitting(true);
    setError(null);
    try {
      await onSubmit(trimmed);
      setValue("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add keyword");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <form onSubmit={handleSubmit} className="flex items-center gap-2">
        <input
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder="e.g. paypal"
          disabled={submitting}
          className="flex-1 min-w-0 rounded-md bg-gray-800 border border-gray-700 px-3 py-2 text-sm
                     placeholder-gray-500 focus:border-blue-500 focus:outline-none focus:ring-1
                     focus:ring-blue-500 disabled:opacity-50"
        />
        <button
          type="submit"
          disabled={submitting || !value.trim()}
          className="shrink-0 rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white
                     hover:bg-blue-700 disabled:opacity-50 transition-colors"
        >
          Add
        </button>
      </form>
      {error && <p className="text-xs text-red-400 mt-1">{error}</p>}
    </div>
  );
}
