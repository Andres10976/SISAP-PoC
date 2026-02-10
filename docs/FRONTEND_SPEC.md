# Frontend Architecture Specification

## Brand Protection Monitor — React Frontend

### 1. Overview

The frontend is a single-page React application providing:

- **Keyword management** — add, view, and remove monitored keywords
- **Monitoring dashboard** — real-time display of matched certificates with color-coded highlighting
- **Status feedback** — live indicator of monitor state and processing metrics
- **CSV export** — one-click download of all matched certificates

**Key technology choices (aligned with job description):**

| Technology   | Version | Purpose                           |
| ------------ | ------- | --------------------------------- |
| React        | 19.x    | UI framework                      |
| TypeScript   | 5.x     | Type safety throughout            |
| Tailwind CSS | 4.x     | Utility-first styling (CSS-first) |
| Vite         | 6.x     | Build tool and dev server         |

> **Migration note:** The project currently uses Create React App (react-scripts).
> This spec targets **Vite**, which is actively maintained, significantly faster,
> and listed in the job description as a desired skill. The migration is a
> prerequisite step before implementing features. (DELETED template and now frontend is an empty foldr)

---

### 2. Project Structure

```
frontend/
├── public/
│   └── favicon.svg
├── src/
│   ├── api/
│   │   ├── client.ts               # Typed fetch wrapper with error handling
│   │   ├── keywords.ts             # Keyword API functions
│   │   ├── certificates.ts         # Certificate API functions
│   │   └── monitor.ts              # Monitor status API functions
│   ├── components/
│   │   ├── layout/
│   │   │   ├── Header.tsx          # App header with monitor status dot
│   │   │   └── Layout.tsx          # Page shell with sidebar + main area
│   │   ├── keywords/
│   │   │   ├── KeywordPanel.tsx    # Keyword management sidebar panel
│   │   │   ├── KeywordForm.tsx     # Input + add button
│   │   │   └── KeywordBadge.tsx    # Removable keyword tag with color
│   │   ├── certificates/
│   │   │   ├── CertificateTable.tsx    # Main data table
│   │   │   ├── CertificateRow.tsx      # Single row with keyword highlighting
│   │   │   ├── EmptyState.tsx          # Empty state when no matches
│   │   │   └── Pagination.tsx          # Page navigation controls
│   │   ├── monitor/
│   │   │   ├── StatusBar.tsx       # Top bar with metrics + controls
│   │   │   └── MetricCard.tsx      # Individual metric display
│   │   └── export/
│   │       └── ExportButton.tsx    # CSV download trigger
│   ├── hooks/
│   │   ├── useKeywords.ts          # Keyword CRUD with optimistic updates
│   │   ├── useCertificates.ts      # Paginated certificate fetching
│   │   ├── useMonitorStatus.ts     # Polling monitor status
│   │   └── usePolling.ts           # Generic polling interval utility
│   ├── types/
│   │   ├── keyword.ts              # Keyword interfaces
│   │   ├── certificate.ts          # Certificate interfaces
│   │   └── monitor.ts              # Monitor state interfaces
│   ├── utils/
│   │   └── colors.ts               # Keyword color assignment
│   ├── App.tsx                     # Root component, composes layout
│   ├── main.tsx                    # Vite entry point
│   └── index.css                   # Tailwind v4 imports + theme
├── index.html                      # Vite HTML entry (at root, not public/)
├── package.json
├── tsconfig.json
├── tsconfig.node.json
├── vite.config.ts
└── Dockerfile
```

---

### 3. Build Configuration

#### 3.1 Vite Configuration

```typescript
// vite.config.ts
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 3000,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
```

The Vite dev server proxies `/api` requests to the Go backend, avoiding CORS
issues during local development.

#### 3.2 TypeScript Configuration

```json
// tsconfig.json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "skipLibCheck": true,
    "isolatedModules": true,
    "esModuleInterop": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": ["src"]
}
```

#### 3.3 Package Dependencies

```json
{
  "dependencies": {
    "react": "^19.2.4",
    "react-dom": "^19.2.4"
  },
  "devDependencies": {
    "@tailwindcss/vite": "^4.1.0",
    "@types/react": "^19.2.0",
    "@types/react-dom": "^19.2.0",
    "@vitejs/plugin-react": "^4.5.0",
    "tailwindcss": "^4.1.0",
    "typescript": "^5.8.0",
    "vite": "^6.3.0"
  }
}
```

> **Intentionally minimal**: No state management library (React hooks are sufficient),
> no router (single-page dashboard), no UI component library (Tailwind utilities
>
> - small custom components).

---

### 4. Type Definitions

All types match the backend API contract exactly.

```typescript
// src/types/keyword.ts

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
```

```typescript
// src/types/certificate.ts

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
```

```typescript
// src/types/monitor.ts

export interface MonitorStatus {
  is_running: boolean;
  last_run_at: string | null;
  last_tree_size: number;
  last_processed_index: number;
  total_processed: number;
  certs_in_last_cycle: number;
  matches_in_last_cycle: number;
  updated_at: string;
}
```

---

### 5. API Client Layer

A typed `fetch` wrapper that centralizes error handling and base URL resolution.

```typescript
// src/api/client.ts

const API_BASE = import.meta.env.VITE_API_URL ?? "/api/v1";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export async function request<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });

  if (!response.ok) {
    const body = await response.json().catch(() => ({
      error: response.statusText,
    }));
    throw new ApiError(response.status, body.error ?? "Request failed");
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}
```

```typescript
// src/api/keywords.ts

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
```

```typescript
// src/api/certificates.ts

import { request } from "./client";
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
  const base = import.meta.env.VITE_API_URL ?? "/api/v1";
  return `${base}/certificates/export`;
}
```

```typescript
// src/api/monitor.ts

import { request } from "./client";
import type { MonitorStatus } from "../types/monitor";

export function fetchMonitorStatus(): Promise<MonitorStatus> {
  return request<MonitorStatus>("/monitor/status");
}

export function startMonitor(): Promise<{ message: string }> {
  return request<{ message: string }>("/monitor/start", { method: "POST" });
}

export function stopMonitor(): Promise<{ message: string }> {
  return request<{ message: string }>("/monitor/stop", { method: "POST" });
}
```

---

### 6. Custom Hooks

#### 6.1 `usePolling` — Generic polling utility

```typescript
// src/hooks/usePolling.ts

import { useEffect, useRef, useCallback, useState } from "react";

export function usePolling<T>(fetcher: () => Promise<T>, intervalMs: number) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const savedFetcher = useRef(fetcher);

  savedFetcher.current = fetcher;

  const refresh = useCallback(async () => {
    try {
      const result = await savedFetcher.current();
      setData(result);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, intervalMs);
    return () => clearInterval(id);
  }, [refresh, intervalMs]);

  return { data, error, loading, refresh };
}
```

#### 6.2 `useKeywords` — Keyword CRUD

```typescript
// src/hooks/useKeywords.ts

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
```

#### 6.3 `useCertificates` — Paginated certificate fetching

```typescript
// src/hooks/useCertificates.ts

import { useState, useEffect, useCallback } from "react";
import * as api from "../api/certificates";
import type { MatchedCertificate } from "../types/certificate";

interface UseCertificatesOptions {
  page: number;
  perPage: number;
  keywordId?: number;
  pollInterval?: number; // ms, 0 to disable
}

export function useCertificates({
  page,
  perPage,
  keywordId,
  pollInterval = 10000,
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
```

#### 6.4 `useMonitorStatus` — Live monitor status

```typescript
// src/hooks/useMonitorStatus.ts

import { useCallback } from "react";
import { usePolling } from "./usePolling";
import * as api from "../api/monitor";
import type { MonitorStatus } from "../types/monitor";

export function useMonitorStatus(pollInterval: number = 5000) {
  const { data, error, loading, refresh } = usePolling<MonitorStatus>(
    api.fetchMonitorStatus,
    pollInterval,
  );

  const start = useCallback(async () => {
    await api.startMonitor();
    refresh();
  }, [refresh]);

  const stop = useCallback(async () => {
    await api.stopMonitor();
    refresh();
  }, [refresh]);

  return { status: data, error, loading, start, stop, refresh };
}
```

---

### 7. Components

#### 7.1 App Root

```tsx
// src/App.tsx

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
```

#### 7.2 Layout

```tsx
// src/components/layout/Layout.tsx

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  return (
    <div className="min-h-screen bg-gray-950 text-gray-100">
      <Header />
      <main className="mx-auto max-w-screen-2xl px-6 py-6 flex flex-col gap-6 h-[calc(100vh-4rem)]">
        {children}
      </main>
    </div>
  );
}
```

```tsx
// src/components/layout/Header.tsx

interface HeaderProps {
  isMonitorRunning?: boolean;
}

export function Header({ isMonitorRunning }: HeaderProps) {
  return (
    <header className="h-16 border-b border-gray-800 bg-gray-900 px-6 flex items-center justify-between">
      <div className="flex items-center gap-3">
        <h1 className="text-lg font-semibold tracking-tight">
          CT Brand Monitor
        </h1>
        <span className="text-xs text-gray-500 font-mono">
          Certificate Transparency
        </span>
      </div>
      {isMonitorRunning !== undefined && (
        <div className="flex items-center gap-2 text-sm">
          <span
            className={`h-2 w-2 rounded-full ${
              isMonitorRunning ? "bg-emerald-400 animate-pulse" : "bg-gray-600"
            }`}
          />
          <span className="text-gray-400">
            {isMonitorRunning ? "Monitoring" : "Stopped"}
          </span>
        </div>
      )}
    </header>
  );
}
```

#### 7.3 Status Bar

```tsx
// src/components/monitor/StatusBar.tsx

import type { MonitorStatus } from "../../types/monitor";
import { MetricCard } from "./MetricCard";
import { ExportButton } from "../export/ExportButton";

interface StatusBarProps {
  status: MonitorStatus | null;
  loading: boolean;
  onStart: () => void;
  onStop: () => void;
}

export function StatusBar({
  status,
  loading,
  onStart,
  onStop,
}: StatusBarProps) {
  const isRunning = status?.is_running ?? false;

  return (
    <div className="flex items-center gap-4 rounded-lg bg-gray-900 border border-gray-800 p-4">
      {/* Start / Stop button */}
      <button
        onClick={isRunning ? onStop : onStart}
        disabled={loading}
        className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
          isRunning
            ? "bg-red-600 hover:bg-red-700 text-white"
            : "bg-emerald-600 hover:bg-emerald-700 text-white"
        } disabled:opacity-50`}
      >
        {isRunning ? "Stop Monitor" : "Start Monitor"}
      </button>

      {/* Metrics */}
      <div className="flex gap-4 flex-1">
        <MetricCard
          label="Total Processed"
          value={status?.total_processed ?? 0}
        />
        <MetricCard
          label="Last Cycle"
          value={status?.certs_in_last_cycle ?? 0}
          suffix="certs"
        />
        <MetricCard
          label="Last Matches"
          value={status?.matches_in_last_cycle ?? 0}
        />
        <MetricCard label="Tree Size" value={status?.last_tree_size ?? 0} />
      </div>

      {/* Last run time */}
      {status?.last_run_at && (
        <span className="text-xs text-gray-500">
          Last run: {new Date(status.last_run_at).toLocaleTimeString()}
        </span>
      )}

      <ExportButton />
    </div>
  );
}
```

```tsx
// src/components/monitor/MetricCard.tsx

interface MetricCardProps {
  label: string;
  value: number;
  suffix?: string;
}

export function MetricCard({ label, value, suffix }: MetricCardProps) {
  return (
    <div className="flex flex-col">
      <span className="text-xs text-gray-500 uppercase tracking-wider">
        {label}
      </span>
      <span className="text-lg font-mono font-semibold tabular-nums">
        {value.toLocaleString()}
        {suffix && <span className="text-xs text-gray-500 ml-1">{suffix}</span>}
      </span>
    </div>
  );
}
```

#### 7.4 Keyword Panel

```tsx
// src/components/keywords/KeywordPanel.tsx

import type { Keyword } from "../../types/keyword";
import { KeywordForm } from "./KeywordForm";
import { KeywordBadge } from "./KeywordBadge";

interface KeywordPanelProps {
  keywords: Keyword[];
  loading: boolean;
  onAdd: (value: string) => Promise<void>;
  onRemove: (id: number) => Promise<void>;
  onFilter: (keywordId: number | undefined) => void;
  activeFilter: number | undefined;
}

export function KeywordPanel({
  keywords,
  loading,
  onAdd,
  onRemove,
  onFilter,
  activeFilter,
}: KeywordPanelProps) {
  return (
    <aside className="w-72 shrink-0 flex flex-col gap-4 rounded-lg bg-gray-900 border border-gray-800 p-4">
      <h2 className="text-sm font-semibold uppercase tracking-wider text-gray-400">
        Monitored Keywords
      </h2>

      <KeywordForm onSubmit={onAdd} />

      {loading ? (
        <p className="text-sm text-gray-500">Loading...</p>
      ) : keywords.length === 0 ? (
        <p className="text-sm text-gray-500">
          No keywords configured. Add one above to start monitoring.
        </p>
      ) : (
        <div className="flex flex-col gap-2 overflow-y-auto">
          {/* "All" filter */}
          <button
            onClick={() => onFilter(undefined)}
            className={`text-left text-sm px-2 py-1 rounded ${
              activeFilter === undefined
                ? "bg-gray-700 text-white"
                : "text-gray-400 hover:text-gray-200"
            }`}
          >
            All keywords
          </button>
          {keywords.map((kw) => (
            <KeywordBadge
              key={kw.id}
              keyword={kw}
              onRemove={() => onRemove(kw.id)}
              onFilter={() => onFilter(kw.id)}
              isActive={activeFilter === kw.id}
            />
          ))}
        </div>
      )}
    </aside>
  );
}
```

```tsx
// src/components/keywords/KeywordForm.tsx

import { useState, type FormEvent } from "react";

interface KeywordFormProps {
  onSubmit: (value: string) => Promise<void>;
}

export function KeywordForm({ onSubmit }: KeywordFormProps) {
  const [value, setValue] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = value.trim();
    if (!trimmed) return;

    setSubmitting(true);
    setError(null);
    try {
      await onSubmit(trimmed);
      setValue("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to add keyword");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex gap-2">
      <input
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder="e.g. paypal"
        disabled={submitting}
        className="flex-1 rounded-md bg-gray-800 border border-gray-700 px-3 py-2 text-sm
                   placeholder-gray-500 focus:border-blue-500 focus:outline-none focus:ring-1
                   focus:ring-blue-500 disabled:opacity-50"
      />
      <button
        type="submit"
        disabled={submitting || !value.trim()}
        className="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white
                   hover:bg-blue-700 disabled:opacity-50 transition-colors"
      >
        Add
      </button>
      {error && <p className="text-xs text-red-400 mt-1">{error}</p>}
    </form>
  );
}
```

```tsx
// src/components/keywords/KeywordBadge.tsx

import type { Keyword } from "../../types/keyword";
import { getKeywordColor } from "../../utils/colors";

interface KeywordBadgeProps {
  keyword: Keyword;
  onRemove: () => void;
  onFilter: () => void;
  isActive: boolean;
}

export function KeywordBadge({
  keyword,
  onRemove,
  onFilter,
  isActive,
}: KeywordBadgeProps) {
  const color = getKeywordColor(keyword.id);

  return (
    <div
      className={`flex items-center justify-between rounded-md px-2 py-1.5
                  border transition-colors cursor-pointer ${color.border} ${
                    isActive ? color.activeBg : "hover:bg-gray-800"
                  }`}
      onClick={onFilter}
    >
      <div className="flex items-center gap-2">
        <span className={`h-2.5 w-2.5 rounded-full ${color.dot}`} />
        <span className="text-sm font-medium">{keyword.value}</span>
      </div>
      <button
        onClick={(e) => {
          e.stopPropagation();
          onRemove();
        }}
        className="text-gray-500 hover:text-red-400 text-xs ml-2"
        aria-label={`Remove ${keyword.value}`}
      >
        ✕
      </button>
    </div>
  );
}
```

#### 7.5 Certificate Table

```tsx
// src/components/certificates/CertificateTable.tsx

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
  keywords,
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
```

```tsx
// src/components/certificates/CertificateRow.tsx

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
```

```tsx
// src/components/certificates/Pagination.tsx

interface PaginationProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}

export function Pagination({
  page,
  totalPages,
  onPageChange,
}: PaginationProps) {
  return (
    <div className="flex items-center gap-2">
      <button
        onClick={() => onPageChange(page - 1)}
        disabled={page <= 1}
        className="rounded px-2 py-1 text-xs bg-gray-800 hover:bg-gray-700
                   disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
      >
        Previous
      </button>
      <span className="text-xs text-gray-500">
        {page} / {totalPages}
      </span>
      <button
        onClick={() => onPageChange(page + 1)}
        disabled={page >= totalPages}
        className="rounded px-2 py-1 text-xs bg-gray-800 hover:bg-gray-700
                   disabled:opacity-30 disabled:cursor-not-allowed transition-colors"
      >
        Next
      </button>
    </div>
  );
}
```

```tsx
// src/components/certificates/EmptyState.tsx

export function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center h-48 text-gray-500 gap-2">
      <p className="text-sm">No matched certificates yet.</p>
      <p className="text-xs">
        Add keywords and start the monitor to begin scanning CT logs.
      </p>
    </div>
  );
}
```

#### 7.6 Export Button

```tsx
// src/components/export/ExportButton.tsx

import { exportCertificatesUrl } from "../../api/certificates";

export function ExportButton() {
  function handleExport() {
    // Direct download via browser navigation — no fetch needed
    window.open(exportCertificatesUrl(), "_blank");
  }

  return (
    <button
      onClick={handleExport}
      className="rounded-md border border-gray-700 bg-gray-800 px-3 py-2
                 text-sm text-gray-300 hover:bg-gray-700 transition-colors"
    >
      Export CSV
    </button>
  );
}
```

---

### 8. Keyword Color Assignment

Deterministic color mapping based on keyword ID. Each keyword gets a consistent
color across the entire UI (badge, row highlight, dot).

```typescript
// src/utils/colors.ts

interface KeywordColors {
  dot: string; // Small indicator dot
  badge: string; // Inline badge (keyword name)
  border: string; // Panel border accent
  activeBg: string; // Active filter background
  rowHighlight: string; // Table row left-border highlight
}

const PALETTE: KeywordColors[] = [
  {
    dot: "bg-red-400",
    badge: "bg-red-500/20 text-red-300",
    border: "border-red-500/30",
    activeBg: "bg-red-500/10",
    rowHighlight: "border-l-2 border-l-red-500",
  },
  {
    dot: "bg-amber-400",
    badge: "bg-amber-500/20 text-amber-300",
    border: "border-amber-500/30",
    activeBg: "bg-amber-500/10",
    rowHighlight: "border-l-2 border-l-amber-500",
  },
  {
    dot: "bg-emerald-400",
    badge: "bg-emerald-500/20 text-emerald-300",
    border: "border-emerald-500/30",
    activeBg: "bg-emerald-500/10",
    rowHighlight: "border-l-2 border-l-emerald-500",
  },
  {
    dot: "bg-sky-400",
    badge: "bg-sky-500/20 text-sky-300",
    border: "border-sky-500/30",
    activeBg: "bg-sky-500/10",
    rowHighlight: "border-l-2 border-l-sky-500",
  },
  {
    dot: "bg-violet-400",
    badge: "bg-violet-500/20 text-violet-300",
    border: "border-violet-500/30",
    activeBg: "bg-violet-500/10",
    rowHighlight: "border-l-2 border-l-violet-500",
  },
  {
    dot: "bg-fuchsia-400",
    badge: "bg-fuchsia-500/20 text-fuchsia-300",
    border: "border-fuchsia-500/30",
    activeBg: "bg-fuchsia-500/10",
    rowHighlight: "border-l-2 border-l-fuchsia-500",
  },
  {
    dot: "bg-cyan-400",
    badge: "bg-cyan-500/20 text-cyan-300",
    border: "border-cyan-500/30",
    activeBg: "bg-cyan-500/10",
    rowHighlight: "border-l-2 border-l-cyan-500",
  },
  {
    dot: "bg-rose-400",
    badge: "bg-rose-500/20 text-rose-300",
    border: "border-rose-500/30",
    activeBg: "bg-rose-500/10",
    rowHighlight: "border-l-2 border-l-rose-500",
  },
];

export function getKeywordColor(keywordId: number): KeywordColors {
  return PALETTE[keywordId % PALETTE.length];
}
```

---

### 9. Tailwind CSS v4 Setup

Tailwind v4 uses a CSS-first configuration model — no `tailwind.config.js`.

```css
/* src/index.css */

@import "tailwindcss";

/* Dark theme customization via @theme */
@theme {
  --color-gray-950: #0a0a0f;
  --color-gray-900: #111118;
  --color-gray-800: #1e1e2a;
  --color-gray-700: #2e2e3e;
  --font-family-mono: "JetBrains Mono", "Fira Code", ui-monospace, monospace;
}

/* Global styles */
body {
  @apply bg-gray-950 text-gray-100 antialiased;
}

/* Scrollbar styling for dark theme */
::-webkit-scrollbar {
  width: 6px;
}
::-webkit-scrollbar-track {
  background: transparent;
}
::-webkit-scrollbar-thumb {
  background: theme(--color-gray-700);
  border-radius: 3px;
}
```

> **Design direction:** Dark theme with a cybersecurity/SOC aesthetic.
> The color palette uses deep blue-grays with high-contrast accent colors
> for keyword-coded certificate highlights. The monospace font is used
> for numeric/technical data (serial numbers, log indices, metrics).

---

### 10. Vite Migration Steps

Since the project currently uses CRA, these are the steps to migrate:

1. Remove CRA dependencies:

   ```
   react-scripts, cra-template-typescript
   @testing-library/* (re-add later with Vitest if needed)
   ```

2. Install Vite dependencies:

   ```
   vite, @vitejs/plugin-react, @tailwindcss/vite
   ```

3. Move `public/index.html` to `index.html` (project root) and add the Vite entry:

   ```html
   <!DOCTYPE html>
   <html lang="en">
     <head>
       <meta charset="UTF-8" />
       <meta name="viewport" content="width=device-width, initial-scale=1.0" />
       <title>CT Brand Monitor</title>
     </head>
     <body>
       <div id="root"></div>
       <script type="module" src="/src/main.tsx"></script>
     </body>
   </html>
   ```

4. Rename `src/index.tsx` to `src/main.tsx`

5. Create `vite.config.ts` and `tsconfig.json` per Section 3

6. Update `package.json` scripts:

   ```json
   {
     "scripts": {
       "dev": "vite",
       "build": "tsc -b && vite build",
       "preview": "vite preview"
     }
   }
   ```

7. Remove `src/reportWebVitals.ts`, `src/setupTests.ts`, `src/App.test.tsx`,
   `src/App.css`, `src/logo.svg`

---

### 11. Dockerfile

Multi-stage build: Vite builds static assets, Nginx serves them.

```dockerfile
# Build stage
FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Serve stage
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html

# Nginx config for SPA + API proxy
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
```

**`nginx.conf`** for production (API proxy to backend service):

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    # SPA fallback
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Proxy API calls to the backend service
    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

---

### 12. UI Layout Wireframe

```
┌──────────────────────────────────────────────────────────────┐
│  CT Brand Monitor                       ● Monitoring         │
├──────────────────────────────────────────────────────────────┤
│ [Start/Stop] │ Total: 1,200 │ Last: 100 │ Matches: 3 │ CSV │
├────────────┬─────────────────────────────────────────────────┤
│ KEYWORDS   │  Domain          Issuer   Keyword  Valid  Disc. │
│            │ ─────────────────────────────────────────────── │
│ + [input]  │ ▌paypal-sec..   R11      paypal   02/09  12:05 │
│            │ ▌goog1e-log..   R10      google   02/09  12:04 │
│ ● paypal   │ ▌paypa1.xyz..   R11      paypal   02/08  12:03 │
│ ● google   │                                                 │
│ ● amazon   │                                                 │
│            │                                                 │
│            │                           Page 1 / 3  [<] [>]  │
└────────────┴─────────────────────────────────────────────────┘
```

- Left sidebar (fixed width 288px): keyword management with colored dots
- Main area: certificate table with keyword-colored left borders on rows
- Top bar: monitor controls + live metrics + export
- Header: app title + live status indicator (green pulsing dot when active)
