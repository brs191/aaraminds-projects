import { getStoredIdentity } from "../identity";
import type { ApiErrorBody } from "./types";

export const API_BASE = "/api/v1";

export class ApiError extends Error {
  status: number;
  code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

interface RequestOptions {
  method?: string;
  json?: unknown;
}

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const identity = getStoredIdentity();
  const headers: Record<string, string> = {
    "X-User-Id": identity.userId,
    "X-User-Role": identity.role,
  };
  let body: string | undefined;
  if (options.json !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.json);
  }

  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      method: options.method ?? "GET",
      headers,
      body,
    });
  } catch {
    throw new ApiError(0, "NETWORK_ERROR", "Could not reach the backend (is it running on :8080?)");
  }

  if (!res.ok) {
    let code = `HTTP_${res.status}`;
    let message = res.statusText || `Request failed with status ${res.status}`;
    try {
      const data = (await res.json()) as Partial<ApiErrorBody>;
      if (data && data.error) {
        code = data.error.code ?? code;
        message = data.error.message ?? message;
      }
    } catch {
      // non-JSON error body; keep defaults
    }
    throw new ApiError(res.status, code, message);
  }

  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

export async function apiForm<T>(path: string, formData: FormData): Promise<T> {
  const identity = getStoredIdentity();
  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      method: "POST",
      headers: {
        "X-User-Id": identity.userId,
        "X-User-Role": identity.role,
      },
      body: formData,
    });
  } catch {
    throw new ApiError(0, "NETWORK_ERROR", "Could not reach the backend (is it running on :8080?)");
  }

  if (!res.ok) {
    let code = `HTTP_${res.status}`;
    let message = res.statusText || `Request failed with status ${res.status}`;
    try {
      const data = (await res.json()) as Partial<ApiErrorBody>;
      if (data && data.error) {
        code = data.error.code ?? code;
        message = data.error.message ?? message;
      }
    } catch {
      // non-JSON error body; keep defaults
    }
    throw new ApiError(res.status, code, message);
  }

  const text = await res.text();
  if (!text) return undefined as T;
  return JSON.parse(text) as T;
}

/** GET that returns null on 404 (summary / quality-report contracts). */
export async function apiMaybe<T>(path: string): Promise<T | null> {
  try {
    return await api<T>(path);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return null;
    throw err;
  }
}

export function exportDownloadUrl(downloadUrl: string): string {
  if (downloadUrl.startsWith("/")) return downloadUrl;
  return `${API_BASE}${downloadUrl}`;
}
