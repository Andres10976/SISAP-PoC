export const API_BASE = import.meta.env.VITE_API_URL ?? "/api/v1";

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
    headers: { ...(options?.body && { "Content-Type": "application/json" }) },
    ...options,
  });

  if (!response.ok) {
    const body = await response.json().catch(() => ({
      error: response.statusText,
    }));
    throw new ApiError(response.status, body.error ?? "Request failed");
  }

  // 204 No Content — callers use void as T for these endpoints
  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}
