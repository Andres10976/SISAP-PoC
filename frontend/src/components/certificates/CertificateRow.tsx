import type { MatchedCertificate } from "../../types/certificate";
import { getKeywordColor } from "../../utils/colors";

interface CertificateRowProps {
  certificate: MatchedCertificate;
}

export function CertificateRow({ certificate: cert }: CertificateRowProps) {
  const color = getKeywordColor(cert.keyword_id);

  return (
    <div
      className={`grid grid-cols-[2fr_1fr_1fr_1fr_1fr] gap-4 px-4 py-3
                  border-b border-gray-800/50 text-sm hover:bg-gray-800/30
                  transition-colors ${color.rowHighlight}`}
    >
      {/* Domain — prominently highlighted */}
      <div className="flex flex-col gap-0.5 min-w-0">
        <span
          className="font-medium text-gray-100 truncate"
          title={cert.matched_domain}
        >
          {cert.matched_domain}
        </span>
        {cert.common_name !== cert.matched_domain && (
          <span
            className="text-xs text-gray-500 truncate"
            title={cert.common_name}
          >
            CN: {cert.common_name}
          </span>
        )}
        {cert.sans.length > 1 && (
          <span className="text-xs text-gray-600">
            +{cert.sans.length - 1} SAN{cert.sans.length > 2 ? "s" : ""}
          </span>
        )}
      </div>

      {/* Issuer */}
      <span className="text-gray-400 truncate" title={cert.issuer}>
        {cert.issuer}
      </span>

      {/* Keyword badge */}
      <div>
        <span
          className={`inline-flex items-center rounded-full px-2.5 py-0.5
                      text-xs font-medium ${color.badge}`}
        >
          {cert.keyword_value}
        </span>
      </div>

      {/* Valid period */}
      <div className="flex flex-col text-xs text-gray-500">
        <span>{new Date(cert.not_before).toLocaleDateString()}</span>
        <span>{new Date(cert.not_after).toLocaleDateString()}</span>
      </div>

      {/* Discovered */}
      <span className="text-gray-500 text-xs">
        {new Date(cert.discovered_at).toLocaleString()}
      </span>
    </div>
  );
}
