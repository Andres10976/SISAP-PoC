import { useState, useEffect, useCallback } from "react";
import * as api from "../api/certificates";
import type { MatchedCertificate } from "../types/certificate";

interface UseCertificatesOptions {
  page: number;
  perPage: number;
  keywordId?: number;
  pollInterval?: number; // ms, 0 to disable
}

// Not using usePolling because this hook re-fetches when page/perPage/keywordId
// change (deps-driven refresh), which usePolling doesn't support.
export function useCertificates({
  page,
  perPage,
  keywordId,
  pollInterval = 65000, // 65s - slightly after backend's 60s CT log poll
}: UseCertificatesOptions) {
  const [certificates, setCertificates] = useState<MatchedCertificate[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    try {
      const data = await api.fetchCertificates(page, perPage, keywordId);
      setCertificates(data.certificates);
      setTotal(data.total);
    } catch {
      // Silent fail on polling — stale data is better than empty
    } finally {
      setLoading(false);
    }
  }, [page, perPage, keywordId]);

  useEffect(() => {
    setLoading(true);
    refresh();

    if (pollInterval > 0) {
      const id = setInterval(refresh, pollInterval);
      return () => clearInterval(id);
    }
  }, [refresh, pollInterval]);

  return { certificates, total, loading, refresh };
}
