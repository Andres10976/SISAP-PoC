import { Layout } from "./components/layout/Layout";
import { StatusBar } from "./components/monitor/StatusBar";
import { KeywordPanel } from "./components/keywords/KeywordPanel";
import { CertificateTable } from "./components/certificates/CertificateTable";
import { useKeywords } from "./hooks/useKeywords";
import { useCertificates } from "./hooks/useCertificates";
import { useMonitorStatus } from "./hooks/useMonitorStatus";
import { useState } from "react";

export default function App() {
  const keywords = useKeywords();
  const monitor = useMonitorStatus();
  const [page, setPage] = useState(1);
  const [filterKeyword, setFilterKeyword] = useState<number | undefined>();

  const certificates = useCertificates({
    page,
    perPage: 20,
    keywordId: filterKeyword,
  });

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
          onAdd={keywords.addKeyword}
          onRemove={keywords.removeKeyword}
          onFilter={setFilterKeyword}
          activeFilter={filterKeyword}
        />
        <CertificateTable
          certificates={certificates.certificates}
          total={certificates.total}
          page={page}
          perPage={20}
          loading={certificates.loading}
          onPageChange={setPage}
          keywords={keywords.keywords}
        />
      </div>
    </Layout>
  );
}
