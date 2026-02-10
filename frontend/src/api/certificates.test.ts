import { fetchCertificates, exportCertificatesUrl } from "./certificates";
import { request } from "./client";

vi.mock("./client", () => ({
  request: vi.fn(),
}));

const mockRequest = vi.mocked(request);

describe("certificates API", () => {
  it("fetchCertificates calls request with page and per_page params", async () => {
    mockRequest.mockResolvedValue({ certificates: [], total: 0, page: 1, per_page: 20 });
    await fetchCertificates(1, 20);
    expect(mockRequest).toHaveBeenCalledWith(
      expect.stringContaining("/certificates?"),
    );
    const url = mockRequest.mock.calls[0][0];
    const params = new URLSearchParams(url.split("?")[1]);
    expect(params.get("page")).toBe("1");
    expect(params.get("per_page")).toBe("20");
  });

  it("includes keyword param when keywordId is provided", async () => {
    mockRequest.mockResolvedValue({ certificates: [], total: 0, page: 1, per_page: 20 });
    await fetchCertificates(2, 10, 5);
    const url = mockRequest.mock.calls[0][0];
    const params = new URLSearchParams(url.split("?")[1]);
    expect(params.get("keyword")).toBe("5");
  });

  it("omits keyword param when keywordId is undefined", async () => {
    mockRequest.mockResolvedValue({ certificates: [], total: 0, page: 1, per_page: 20 });
    await fetchCertificates(1, 20);
    const url = mockRequest.mock.calls[0][0];
    const params = new URLSearchParams(url.split("?")[1]);
    expect(params.has("keyword")).toBe(false);
  });

  it("uses default values for page and perPage", async () => {
    mockRequest.mockResolvedValue({ certificates: [], total: 0, page: 1, per_page: 20 });
    await fetchCertificates();
    const url = mockRequest.mock.calls[0][0];
    const params = new URLSearchParams(url.split("?")[1]);
    expect(params.get("page")).toBe("1");
    expect(params.get("per_page")).toBe("20");
  });

  it("exportCertificatesUrl returns the default export URL", () => {
    expect(exportCertificatesUrl()).toBe("/api/v1/certificates/export");
  });

  it("exportCertificatesUrl uses VITE_API_URL env when set", () => {
    vi.stubEnv("VITE_API_URL", "https://api.example.com");
    expect(exportCertificatesUrl()).toBe("https://api.example.com/certificates/export");
    vi.unstubAllEnvs();
  });
});
