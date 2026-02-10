import { Layout } from "./components/layout/Layout";
import { StatusBar } from "./components/monitor/StatusBar";
import { KeywordPanel } from "./components/keywords/KeywordPanel";
import { CertificateTable } from "./components/certificates/CertificateTable";
import { useKeywords } from "./hooks/useKeywords";
import { useCertificates } from "./hooks/useCertificates";
import { useMonitorStatus } from "./hooks/useMonitorStatus";
import { useCallback, useState, useEffect, useRef } from "react";

export default function App() {
  const keywords = useKeywords();
  const monitor = useMonitorStatus();
  const [page, setPage] = useState(1);
  const [filterKeyword, setFilterKeyword] = useState<number | undefined>();

  const handleFilterChange = useCallback((keywordId: number | undefined) => {
    setFilterKeyword(keywordId);
    setPage(1);
  }, []);

  const certificates = useCertificates({
    page,
    perPage: 20,
    keywordId: filterKeyword,
  });

  // Auto-refresh certificates when monitor completes a batch with new matches
  const prevLastRunRef = useRef<string | null>(null);
  const refreshRef = useRef(certificates.refresh);
  refreshRef.current = certificates.refresh;

  useEffect(() => {
    const lastRun = monitor.status?.last_run_at;
    const prevLastRun = prevLastRunRef.current;
    const matchesInLastCycle = monitor.status?.matches_in_last_cycle ?? 0;

    // When last_run_at changes to a non-null value and there are matches, refresh certificates
    if (lastRun !== prevLastRun && lastRun !== null && matchesInLastCycle > 0) {
      refreshRef.current();
    }

    prevLastRunRef.current = lastRun ?? null;
  }, [monitor.status?.last_run_at, monitor.status?.matches_in_last_cycle]);

  const handleAddKeyword = useCallback(
    async (value: string) => {
      await keywords.addKeyword(value);
      certificates.refresh();
    },
    [keywords.addKeyword, certificates.refresh],
  );

  const handleRemoveKeyword = useCallback(
    async (id: number) => {
      await keywords.removeKeyword(id);
      certificates.refresh();
    },
    [keywords.removeKeyword, certificates.refresh],
  );

  return (
    <Layout>
      <StatusBar
        status={monitor.status}
        loading={monitor.loading}
        onStart={monitor.start}
        onStop={monitor.stop}
      />
      <div className="flex gap-6 flex-1 min-h-0">
        <KeywordPanel
          keywords={keywords.keywords}
          loading={keywords.loading}
          onAdd={handleAddKeyword}
          onRemove={handleRemoveKeyword}
          onFilter={handleFilterChange}
          activeFilter={filterKeyword}
        />
        <CertificateTable
          certificates={certificates.certificates}
          total={certificates.total}
          page={page}
          perPage={20}
          loading={certificates.loading}
          onPageChange={setPage}
        />
      </div>
    </Layout>
  );
}
