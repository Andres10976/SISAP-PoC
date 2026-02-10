import { renderHook, act } from "@testing-library/react";
import { useMonitorStatus } from "./useMonitorStatus";
import * as api from "../api/monitor";

vi.mock("../api/monitor", () => ({
  fetchMonitorStatus: vi.fn(),
  startMonitor: vi.fn(),
  stopMonitor: vi.fn(),
}));

const mockFetchStatus = vi.mocked(api.fetchMonitorStatus);
const mockStart = vi.mocked(api.startMonitor);
const mockStop = vi.mocked(api.stopMonitor);

const mockStatus = {
  is_running: true,
  last_run_at: "2024-01-01T00:00:00Z",
  last_tree_size: 1000,
  last_processed_index: 500,
  total_processed: 5000,
  certs_in_last_cycle: 100,
  matches_in_last_cycle: 3,
  updated_at: "2024-01-01T12:00:00Z",
};

describe("useMonitorStatus", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("fetches monitor status on mount via usePolling", async () => {
    mockFetchStatus.mockResolvedValue(mockStatus);
    const { result } = renderHook(() => useMonitorStatus(5000));
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(result.current.status).toEqual(mockStatus);
  });

  it("starts in loading state", () => {
    mockFetchStatus.mockResolvedValue(mockStatus);
    const { result } = renderHook(() => useMonitorStatus(5000));
    expect(result.current.loading).toBe(true);
  });

  it("exposes error from usePolling", async () => {
    mockFetchStatus.mockRejectedValue(new Error("connection refused"));
    const { result } = renderHook(() => useMonitorStatus(5000));
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(result.current.error).toBe("connection refused");
  });

  it("start calls startMonitor and then refreshes", async () => {
    mockFetchStatus.mockResolvedValue(mockStatus);
    mockStart.mockResolvedValue({ message: "started" });

    const { result } = renderHook(() => useMonitorStatus(60000));
    await act(() => vi.advanceTimersByTimeAsync(0));
    const callsBefore = mockFetchStatus.mock.calls.length;

    await act(() => result.current.start());
    expect(mockStart).toHaveBeenCalled();
    expect(mockFetchStatus.mock.calls.length).toBeGreaterThan(callsBefore);
  });

  it("stop calls stopMonitor and then refreshes", async () => {
    mockFetchStatus.mockResolvedValue(mockStatus);
    mockStop.mockResolvedValue({ message: "stopped" });

    const { result } = renderHook(() => useMonitorStatus(60000));
    await act(() => vi.advanceTimersByTimeAsync(0));
    const callsBefore = mockFetchStatus.mock.calls.length;

    await act(() => result.current.stop());
    expect(mockStop).toHaveBeenCalled();
    expect(mockFetchStatus.mock.calls.length).toBeGreaterThan(callsBefore);
  });

  it("polls at the specified interval", async () => {
    mockFetchStatus.mockResolvedValue(mockStatus);
    renderHook(() => useMonitorStatus(3000));
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(mockFetchStatus).toHaveBeenCalledTimes(1);

    await act(() => vi.advanceTimersByTimeAsync(3000));
    expect(mockFetchStatus).toHaveBeenCalledTimes(2);
  });

  it("uses default 5000ms poll interval", async () => {
    mockFetchStatus.mockResolvedValue(mockStatus);
    renderHook(() => useMonitorStatus());
    await act(() => vi.advanceTimersByTimeAsync(0));

    await act(() => vi.advanceTimersByTimeAsync(5000));
    expect(mockFetchStatus).toHaveBeenCalledTimes(2);
  });
});
