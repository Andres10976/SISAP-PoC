import { fetchMonitorStatus, startMonitor, stopMonitor } from "./monitor";
import { request } from "./client";

vi.mock("./client", () => ({
  request: vi.fn(),
}));

const mockRequest = vi.mocked(request);

describe("monitor API", () => {
  it("fetchMonitorStatus calls request with /monitor/status", async () => {
    mockRequest.mockResolvedValue({ is_running: false });
    await fetchMonitorStatus();
    expect(mockRequest).toHaveBeenCalledWith("/monitor/status");
  });

  it("startMonitor sends POST to /monitor/start", async () => {
    mockRequest.mockResolvedValue({ message: "started" });
    const result = await startMonitor();
    expect(mockRequest).toHaveBeenCalledWith("/monitor/start", { method: "POST" });
    expect(result).toEqual({ message: "started" });
  });

  it("stopMonitor sends POST to /monitor/stop", async () => {
    mockRequest.mockResolvedValue({ message: "stopped" });
    const result = await stopMonitor();
    expect(mockRequest).toHaveBeenCalledWith("/monitor/stop", { method: "POST" });
    expect(result).toEqual({ message: "stopped" });
  });
});
