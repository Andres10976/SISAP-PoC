# Frontend — CT Brand Monitor

React + TypeScript SPA for the Certificate Transparency Brand Monitor PoC.

## Tech Stack

- **React 19** with functional components and hooks
- **TypeScript** (strict mode, `noUnusedLocals`, `noUnusedParameters`)
- **Vite 6** for dev server and bundling
- **Tailwind CSS v4** via `@tailwindcss/vite` plugin
- **Vitest 4** with `jsdom` environment and `@testing-library/react`

## Commands

```bash
npm run dev        # Start dev server on :3000 (proxies /api → localhost:8080)
npm run build      # Type-check (tsc) then build for production
npm run preview    # Preview production build
npm run test       # Run tests in watch mode
npm run test:run   # Run tests once (CI)
```

## Project Structure

```
src/
  api/            # HTTP client and endpoint modules (client.ts, keywords.ts, certificates.ts, monitor.ts)
  components/     # UI components grouped by feature
    layout/       #   Header, Layout (app shell)
    keywords/     #   KeywordForm, KeywordBadge, KeywordPanel
    certificates/ #   CertificateRow, CertificateTable, Pagination, EmptyState
    monitor/      #   MetricCard, StatusBar
    export/       #   ExportButton
  hooks/          # Custom React hooks (useKeywords, useCertificates, useMonitorStatus, usePolling)
  types/          # TypeScript interfaces (keyword.ts, certificate.ts, monitor.ts)
  utils/          # Helpers (colors.ts)
  test/           # Test setup (global fetch stub)
  App.tsx         # Root component
  main.tsx        # Entry point
  index.css       # Tailwind import + dark theme overrides
```

## Architecture Patterns

- **API layer**: `src/api/client.ts` exports a generic `request<T>()` function using `fetch`. All endpoint modules (`keywords.ts`, `certificates.ts`, `monitor.ts`) build on it. Base URL comes from `VITE_API_URL` env var, defaults to `/api/v1`.
- **Custom hooks**: Each feature has a hook that owns state (`useState`), fetches data (`useEffect`/`useCallback`), and returns data + actions. Hooks are the bridge between API and components.
- **Components**: Functional components only. Props-driven, no internal data fetching — all state comes from hooks in `App.tsx`.
- **Dark theme**: Custom gray palette defined in `index.css` via Tailwind `@theme`. Body defaults to `bg-gray-950 text-gray-100`.

## Path Aliases

`@/*` maps to `src/*`. Configured in both `tsconfig.json` and `vitest.config.ts`.

```tsx
import { useKeywords } from "@/hooks/useKeywords";
```

## Conventions

- **Files**: `kebab-case` for directories, `PascalCase` for components (`KeywordPanel.tsx`), `camelCase` for hooks/utils/api modules (`useKeywords.ts`, `client.ts`).
- **Types**: Interfaces for data shapes, exported from `src/types/`. Use `type` imports (`import type { Keyword }`).
- **Exports**: Named exports for components and hooks (no default exports except `App.tsx`).
- **No `any`**: TypeScript strict mode is enforced. Avoid `any`.
- **Tests**: Co-located with source files (`*.test.ts` next to the module). Use `vi.mock()` for module mocking and `vi.mocked()` for typed access. `fetch` is globally stubbed in `src/test/setup.ts` — each test must provide its own mock implementation.
- **Vitest globals**: `describe`, `it`, `expect`, `vi` are available globally (no imports needed).

## Environment Variables

| Variable | Purpose | Default |
|---|---|---|
| `VITE_API_URL` | Backend API base URL | `/api/v1` |

## Docker

Multi-stage build: `node:22-alpine` for build, `nginx:alpine` for serving. Nginx config (`nginx.conf`) handles SPA fallback and proxies `/api/` to `http://backend:8080`.
