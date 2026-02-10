import { renderHook, act } from "@testing-library/react";
import { useCertificates } from "./useCertificates";
import * as api from "../api/certificates";

vi.mock("../api/certificates", () => ({
  fetchCertificates: vi.fn(),
  exportCertificatesUrl: vi.fn(),
}));

const mockFetchCertificates = vi.mocked(api.fetchCertificates);

const cert1 = {
  id: 1,
  serial_number: "AA:BB",
  common_name: "example.com",
  sans: ["example.com"],
  issuer: "Let's Encrypt",
  not_before: "2024-01-01T00:00:00Z",
  not_after: "2024-12-31T23:59:59Z",
  keyword_id: 1,
  keyword_value: "example",
  matched_domain: "example.com",
  ct_log_index: 100,
  discovered_at: "2024-01-01T12:00:00Z",
};

describe("useCertificates", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("fetches certificates on mount", async () => {
    mockFetchCertificates.mockResolvedValue({
      certificates: [cert1],
      total: 1,
      page: 1,
      per_page: 20,
    });
    const { result } = renderHook(() =>
      useCertificates({ page: 1, perPage: 20 }),
    );
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(result.current.certificates).toEqual([cert1]);
    expect(result.current.total).toBe(1);
  });

  it("passes page, perPage and keywordId to fetchCertificates", async () => {
    mockFetchCertificates.mockResolvedValue({
      certificates: [],
      total: 0,
      page: 2,
      per_page: 10,
    });
    renderHook(() =>
      useCertificates({ page: 2, perPage: 10, keywordId: 5 }),
    );
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(mockFetchCertificates).toHaveBeenCalledWith(2, 10, 5);
  });

  it("starts loading=true and sets to false after fetch", async () => {
    mockFetchCertificates.mockResolvedValue({
      certificates: [],
      total: 0,
      page: 1,
      per_page: 20,
    });
    const { result } = renderHook(() =>
      useCertificates({ page: 1, perPage: 20 }),
    );
    expect(result.current.loading).toBe(true);
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(result.current.loading).toBe(false);
  });

  it("silently handles errors (does not expose error state)", async () => {
    mockFetchCertificates.mockRejectedValue(new Error("server error"));
    const { result } = renderHook(() =>
      useCertificates({ page: 1, perPage: 20 }),
    );
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(result.current.loading).toBe(false);
    expect(result.current.certificates).toEqual([]);
  });

  it("polls at the specified interval", async () => {
    mockFetchCertificates.mockResolvedValue({
      certificates: [],
      total: 0,
      page: 1,
      per_page: 20,
    });
    renderHook(() =>
      useCertificates({ page: 1, perPage: 20, pollInterval: 3000 }),
    );
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(mockFetchCertificates).toHaveBeenCalledTimes(1);

    await act(() => vi.advanceTimersByTimeAsync(3000));
    expect(mockFetchCertificates).toHaveBeenCalledTimes(2);
  });

  it("does not poll when pollInterval is 0", async () => {
    mockFetchCertificates.mockResolvedValue({
      certificates: [],
      total: 0,
      page: 1,
      per_page: 20,
    });
    renderHook(() =>
      useCertificates({ page: 1, perPage: 20, pollInterval: 0 }),
    );
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(mockFetchCertificates).toHaveBeenCalledTimes(1);

    await act(() => vi.advanceTimersByTimeAsync(30000));
    expect(mockFetchCertificates).toHaveBeenCalledTimes(1);
  });

  it("re-fetches when page changes", async () => {
    mockFetchCertificates.mockResolvedValue({
      certificates: [],
      total: 0,
      page: 1,
      per_page: 20,
    });
    const { rerender } = renderHook(
      (props) => useCertificates(props),
      { initialProps: { page: 1, perPage: 20, pollInterval: 0 } },
    );
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(mockFetchCertificates).toHaveBeenCalledTimes(1);

    rerender({ page: 2, perPage: 20, pollInterval: 0 });
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(mockFetchCertificates).toHaveBeenCalledTimes(2);
  });

  it("re-fetches when keywordId changes", async () => {
    mockFetchCertificates.mockResolvedValue({
      certificates: [],
      total: 0,
      page: 1,
      per_page: 20,
    });
    const { rerender } = renderHook(
      (props) => useCertificates(props),
      { initialProps: { page: 1, perPage: 20, keywordId: 1, pollInterval: 0 } as { page: number; perPage: number; keywordId?: number; pollInterval: number } },
    );
    await act(() => vi.advanceTimersByTimeAsync(0));

    rerender({ page: 1, perPage: 20, keywordId: 2, pollInterval: 0 });
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(mockFetchCertificates).toHaveBeenCalledTimes(2);
  });

  it("exposes refresh function for manual reload", async () => {
    mockFetchCertificates.mockResolvedValue({
      certificates: [cert1],
      total: 1,
      page: 1,
      per_page: 20,
    });
    const { result } = renderHook(() =>
      useCertificates({ page: 1, perPage: 20, pollInterval: 0 }),
    );
    await act(() => vi.advanceTimersByTimeAsync(0));

    mockFetchCertificates.mockResolvedValue({
      certificates: [],
      total: 0,
      page: 1,
      per_page: 20,
    });
    await act(() => result.current.refresh());
    expect(result.current.certificates).toEqual([]);
  });
});
