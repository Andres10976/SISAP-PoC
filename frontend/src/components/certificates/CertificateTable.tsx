import type { MatchedCertificate } from "../../types/certificate";
import type { Keyword } from "../../types/keyword";
import { CertificateRow } from "./CertificateRow";
import { EmptyState } from "./EmptyState";
import { Pagination } from "./Pagination";

interface CertificateTableProps {
  certificates: MatchedCertificate[];
  total: number;
  page: number;
  perPage: number;
  loading: boolean;
  onPageChange: (page: number) => void;
  keywords: Keyword[];
}

export function CertificateTable({
  certificates,
  total,
  page,
  perPage,
  loading,
  onPageChange,
  keywords: _keywords,
}: CertificateTableProps) {
  const totalPages = Math.ceil(total / perPage);

  return (
    <div className="flex-1 flex flex-col rounded-lg bg-gray-900 border border-gray-800 overflow-hidden">
      {/* Table header */}
      <div
        className="grid grid-cols-[2fr_1fr_1fr_1fr_1fr] gap-4 px-4 py-3
                      border-b border-gray-800 text-xs text-gray-500
                      uppercase tracking-wider font-medium"
      >
        <span>Domain</span>
        <span>Issuer</span>
        <span>Keyword</span>
        <span>Valid Period</span>
        <span>Discovered</span>
      </div>

      {/* Table body */}
      <div className="flex-1 overflow-y-auto">
        {loading && certificates.length === 0 ? (
          <div className="flex items-center justify-center h-32 text-gray-500 text-sm">
            Loading certificates...
          </div>
        ) : certificates.length === 0 ? (
          <EmptyState />
        ) : (
          certificates.map((cert) => (
            <CertificateRow key={cert.id} certificate={cert} />
          ))
        )}
      </div>

      {/* Footer with pagination and total */}
      {total > 0 && (
        <div
          className="flex items-center justify-between px-4 py-3
                        border-t border-gray-800 text-sm text-gray-400"
        >
          <span>
            {total} matched certificate{total !== 1 ? "s" : ""}
          </span>
          <Pagination
            page={page}
            totalPages={totalPages}
            onPageChange={onPageChange}
          />
        </div>
      )}
    </div>
  );
}
