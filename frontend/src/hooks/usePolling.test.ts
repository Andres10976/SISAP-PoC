import { renderHook, act } from "@testing-library/react";
import { usePolling } from "./usePolling";

describe("usePolling", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("calls fetcher immediately on mount", async () => {
    const fetcher = vi.fn().mockResolvedValue({ value: 1 });
    renderHook(() => usePolling(fetcher, 5000));
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it("returns loading=true initially", () => {
    const fetcher = vi.fn().mockResolvedValue({ value: 1 });
    const { result } = renderHook(() => usePolling(fetcher, 5000));
    expect(result.current.loading).toBe(true);
  });

  it("sets data after fetcher resolves", async () => {
    const fetcher = vi.fn().mockResolvedValue({ value: 42 });
    const { result } = renderHook(() => usePolling(fetcher, 5000));
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(result.current.data).toEqual({ value: 42 });
    expect(result.current.loading).toBe(false);
  });

  it("sets error when fetcher rejects", async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error("network fail"));
    const { result } = renderHook(() => usePolling(fetcher, 5000));
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(result.current.error).toBe("network fail");
    expect(result.current.data).toBeNull();
  });

  it("polls at the given interval", async () => {
    const fetcher = vi.fn().mockResolvedValue({ value: 1 });
    renderHook(() => usePolling(fetcher, 3000));

    // Initial call
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(fetcher).toHaveBeenCalledTimes(1);

    // After one interval
    await act(() => vi.advanceTimersByTimeAsync(3000));
    expect(fetcher).toHaveBeenCalledTimes(2);

    // After another interval
    await act(() => vi.advanceTimersByTimeAsync(3000));
    expect(fetcher).toHaveBeenCalledTimes(3);
  });

  it("clears interval on unmount", async () => {
    const fetcher = vi.fn().mockResolvedValue({ value: 1 });
    const { unmount } = renderHook(() => usePolling(fetcher, 3000));
    await act(() => vi.advanceTimersByTimeAsync(0));
    unmount();

    const callsAtUnmount = fetcher.mock.calls.length;
    await act(() => vi.advanceTimersByTimeAsync(10000));
    expect(fetcher).toHaveBeenCalledTimes(callsAtUnmount);
  });

  it("uses the latest fetcher reference via ref", async () => {
    const fetcher1 = vi.fn().mockResolvedValue("first");
    const fetcher2 = vi.fn().mockResolvedValue("second");

    const { rerender, result } = renderHook(
      ({ fn }) => usePolling(fn, 5000),
      { initialProps: { fn: fetcher1 } },
    );
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(result.current.data).toBe("first");

    rerender({ fn: fetcher2 });
    await act(() => vi.advanceTimersByTimeAsync(5000));
    expect(fetcher2).toHaveBeenCalled();
    expect(result.current.data).toBe("second");
  });

  it("clears error on successful fetch after failure", async () => {
    const fetcher = vi.fn()
      .mockRejectedValueOnce(new Error("fail"))
      .mockResolvedValue({ value: "ok" });

    const { result } = renderHook(() => usePolling(fetcher, 5000));
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(result.current.error).toBe("fail");

    await act(() => vi.advanceTimersByTimeAsync(5000));
    expect(result.current.error).toBeNull();
    expect(result.current.data).toEqual({ value: "ok" });
  });

  it("exposes a refresh function that can be called manually", async () => {
    const fetcher = vi.fn().mockResolvedValue({ count: 1 });
    const { result } = renderHook(() => usePolling(fetcher, 60000));
    await act(() => vi.advanceTimersByTimeAsync(0));
    expect(fetcher).toHaveBeenCalledTimes(1);

    fetcher.mockResolvedValue({ count: 2 });
    await act(() => result.current.refresh());
    expect(result.current.data).toEqual({ count: 2 });
  });
});
