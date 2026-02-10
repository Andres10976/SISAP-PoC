import { request, API_BASE } from "./client";
import type { CertificatesResponse } from "../types/certificate";

export function fetchCertificates(
  page: number = 1,
  perPage: number = 20,
  keywordId?: number,
): Promise<CertificatesResponse> {
  const params = new URLSearchParams({
    page: String(page),
    per_page: String(perPage),
  });
  if (keywordId !== undefined) {
    params.set("keyword", String(keywordId));
  }
  return request<CertificatesResponse>(`/certificates?${params}`);
}

export function exportCertificatesUrl(): string {
  return `${API_BASE}/certificates/export`;
}
