import { useState, useEffect, useCallback } from "react";
import * as api from "../api/keywords";
import type { Keyword } from "../types/keyword";

export function useKeywords() {
  const [keywords, setKeywords] = useState<Keyword[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const { keywords } = await api.fetchKeywords();
      setKeywords(keywords);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load keywords");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const addKeyword = useCallback(async (value: string) => {
    const keyword = await api.createKeyword({ value });
    setKeywords((prev) => [keyword, ...prev]);
  }, []);

  const removeKeyword = useCallback(async (id: number) => {
    await api.deleteKeyword(id);
    setKeywords((prev) => prev.filter((kw) => kw.id !== id));
  }, []);

  return { keywords, loading, error, addKeyword, removeKeyword, refresh };
}
