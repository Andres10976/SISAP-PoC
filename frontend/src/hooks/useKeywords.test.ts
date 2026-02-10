import { renderHook, act, waitFor } from "@testing-library/react";
import { useKeywords } from "./useKeywords";
import * as api from "../api/keywords";

vi.mock("../api/keywords", () => ({
  fetchKeywords: vi.fn(),
  createKeyword: vi.fn(),
  deleteKeyword: vi.fn(),
}));

const mockFetchKeywords = vi.mocked(api.fetchKeywords);
const mockCreateKeyword = vi.mocked(api.createKeyword);
const mockDeleteKeyword = vi.mocked(api.deleteKeyword);

const kw1 = { id: 1, value: "google", created_at: "2024-01-01T00:00:00Z" };
const kw2 = { id: 2, value: "amazon", created_at: "2024-01-02T00:00:00Z" };

describe("useKeywords", () => {
  it("fetches keywords on mount", async () => {
    mockFetchKeywords.mockResolvedValue({ keywords: [kw1, kw2] });
    const { result } = renderHook(() => useKeywords());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.keywords).toEqual([kw1, kw2]);
  });

  it("sets error when fetchKeywords fails", async () => {
    mockFetchKeywords.mockRejectedValue(new Error("server down"));
    const { result } = renderHook(() => useKeywords());
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.error).toBe("server down");
  });

  it("addKeyword prepends the new keyword to state", async () => {
    mockFetchKeywords.mockResolvedValue({ keywords: [kw1] });
    const newKw = { id: 3, value: "meta", created_at: "2024-01-03T00:00:00Z" };
    mockCreateKeyword.mockResolvedValue(newKw);

    const { result } = renderHook(() => useKeywords());
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(() => result.current.addKeyword("meta"));
    expect(mockCreateKeyword).toHaveBeenCalledWith({ value: "meta" });
    expect(result.current.keywords[0]).toEqual(newKw);
  });

  it("removeKeyword filters the keyword from state", async () => {
    mockFetchKeywords.mockResolvedValue({ keywords: [kw1, kw2] });
    mockDeleteKeyword.mockResolvedValue(undefined);

    const { result } = renderHook(() => useKeywords());
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(() => result.current.removeKeyword(1));
    expect(mockDeleteKeyword).toHaveBeenCalledWith(1);
    expect(result.current.keywords).toEqual([kw2]);
  });

  it("addKeyword propagates API errors", async () => {
    mockFetchKeywords.mockResolvedValue({ keywords: [] });
    mockCreateKeyword.mockRejectedValue(new Error("duplicate"));

    const { result } = renderHook(() => useKeywords());
    await waitFor(() => expect(result.current.loading).toBe(false));

    await expect(act(() => result.current.addKeyword("dup"))).rejects.toThrow("duplicate");
  });

  it("removeKeyword propagates API errors", async () => {
    mockFetchKeywords.mockResolvedValue({ keywords: [kw1] });
    mockDeleteKeyword.mockRejectedValue(new Error("not found"));

    const { result } = renderHook(() => useKeywords());
    await waitFor(() => expect(result.current.loading).toBe(false));

    await expect(act(() => result.current.removeKeyword(1))).rejects.toThrow("not found");
  });

  it("refresh reloads keywords from the API", async () => {
    mockFetchKeywords.mockResolvedValue({ keywords: [kw1] });
    const { result } = renderHook(() => useKeywords());
    await waitFor(() => expect(result.current.loading).toBe(false));

    mockFetchKeywords.mockResolvedValue({ keywords: [kw1, kw2] });
    await act(() => result.current.refresh());
    expect(result.current.keywords).toEqual([kw1, kw2]);
  });
});
