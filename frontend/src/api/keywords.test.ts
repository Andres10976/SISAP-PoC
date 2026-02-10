import { fetchKeywords, createKeyword, deleteKeyword } from "./keywords";
import { request } from "./client";

vi.mock("./client", () => ({
  request: vi.fn(),
}));

const mockRequest = vi.mocked(request);

describe("keywords API", () => {
  it("fetchKeywords calls request with /keywords", async () => {
    mockRequest.mockResolvedValue({ keywords: [] });
    await fetchKeywords();
    expect(mockRequest).toHaveBeenCalledWith("/keywords");
  });

  it("createKeyword sends POST with body", async () => {
    const keyword = { id: 1, value: "test", created_at: "2024-01-01T00:00:00Z" };
    mockRequest.mockResolvedValue(keyword);
    const result = await createKeyword({ value: "test" });
    expect(mockRequest).toHaveBeenCalledWith("/keywords", {
      method: "POST",
      body: JSON.stringify({ value: "test" }),
    });
    expect(result).toEqual(keyword);
  });

  it("deleteKeyword sends DELETE with id in path", async () => {
    mockRequest.mockResolvedValue(undefined);
    await deleteKeyword(42);
    expect(mockRequest).toHaveBeenCalledWith("/keywords/42", { method: "DELETE" });
  });

  it("fetchKeywords returns the response from request", async () => {
    const response = { keywords: [{ id: 1, value: "test", created_at: "2024-01-01T00:00:00Z" }] };
    mockRequest.mockResolvedValue(response);
    const result = await fetchKeywords();
    expect(result).toEqual(response);
  });

  it("createKeyword propagates request errors", async () => {
    mockRequest.mockRejectedValue(new Error("network error"));
    await expect(createKeyword({ value: "fail" })).rejects.toThrow("network error");
  });
});
