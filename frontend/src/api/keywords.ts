import { request } from "./client";
import type {
  Keyword,
  KeywordsResponse,
  CreateKeywordRequest,
} from "../types/keyword";

export function fetchKeywords(): Promise<KeywordsResponse> {
  return request<KeywordsResponse>("/keywords");
}

export function createKeyword(data: CreateKeywordRequest): Promise<Keyword> {
  return request<Keyword>("/keywords", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export function deleteKeyword(id: number): Promise<void> {
  return request<void>(`/keywords/${id}`, { method: "DELETE" });
}
