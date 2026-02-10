import { request, ApiError } from "./client";

function mockFetch(response: Partial<Response>) {
  const fn = vi.fn<() => Promise<Partial<Response>>>().mockResolvedValue({
    ok: true,
    status: 200,
    json: () => Promise.resolve({}),
    statusText: "OK",
    ...response,
  });
  vi.stubGlobal("fetch", fn);
  return fn;
}

describe("request", () => {
  it("calls fetch with the correct URL", async () => {
    const fetchMock = mockFetch({ json: () => Promise.resolve({ id: 1 }) });
    await request("/keywords");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/keywords",
      expect.objectContaining({ headers: {} }),
    );
  });

  it("sets Content-Type header when body is provided", async () => {
    const fetchMock = mockFetch({ json: () => Promise.resolve({}) });
    await request("/keywords", { method: "POST", body: '{"value":"test"}' });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/keywords",
      expect.objectContaining({
        headers: { "Content-Type": "application/json" },
      }),
    );
  });

  it("returns parsed JSON on success", async () => {
    mockFetch({ json: () => Promise.resolve({ id: 1, value: "test" }) });
    const data = await request<{ id: number; value: string }>("/keywords");
    expect(data).toEqual({ id: 1, value: "test" });
  });

  it("returns undefined for 204 No Content", async () => {
    mockFetch({ status: 204, json: () => Promise.reject(new Error("no body")) });
    const data = await request<void>("/keywords/1");
    expect(data).toBeUndefined();
  });

  it("throws ApiError on non-ok response with error body", async () => {
    mockFetch({
      ok: false,
      status: 422,
      statusText: "Unprocessable Entity",
      json: () => Promise.resolve({ error: "Keyword already exists" }),
    });
    await expect(request("/keywords")).rejects.toThrow(ApiError);
    await expect(request("/keywords")).rejects.toThrow("Keyword already exists");
  });

  it("throws ApiError with statusText when JSON parse fails", async () => {
    mockFetch({
      ok: false,
      status: 500,
      statusText: "Internal Server Error",
      json: () => Promise.reject(new Error("not json")),
    });
    await expect(request("/test")).rejects.toThrow("Internal Server Error");
  });

  it("throws ApiError with 'Request failed' when no error field in body", async () => {
    mockFetch({
      ok: false,
      status: 400,
      statusText: "Bad Request",
      json: () => Promise.resolve({ detail: "something" }),
    });
    await expect(request("/test")).rejects.toThrow("Request failed");
  });

  it("sets the ApiError status property", async () => {
    mockFetch({
      ok: false,
      status: 404,
      statusText: "Not Found",
      json: () => Promise.resolve({ error: "not found" }),
    });
    try {
      await request("/missing");
      expect.unreachable("should have thrown");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).status).toBe(404);
    }
  });

  it("merges caller-provided options with defaults", async () => {
    const fetchMock = mockFetch({ json: () => Promise.resolve({}) });
    await request("/keywords", { method: "POST", body: '{"value":"test"}' });
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/keywords",
      expect.objectContaining({ method: "POST", body: '{"value":"test"}' }),
    );
  });
});
