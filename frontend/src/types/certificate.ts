export interface MatchedCertificate {
  id: number;
  serial_number: string;
  common_name: string;
  sans: string[];
  issuer: string;
  not_before: string; // ISO 8601
  not_after: string; // ISO 8601
  keyword_id: number;
  keyword_value: string;
  matched_domain: string;
  ct_log_index: number;
  discovered_at: string; // ISO 8601
}

export interface CertificatesResponse {
  certificates: MatchedCertificate[];
  total: number;
  page: number;
  per_page: number;
}
