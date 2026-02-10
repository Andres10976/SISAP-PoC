export interface Keyword {
  id: number;
  value: string;
  created_at: string; // ISO 8601
}

export interface KeywordsResponse {
  keywords: Keyword[];
}

export interface CreateKeywordRequest {
  value: string;
}
