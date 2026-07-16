"use client";

import { ApiResponse } from "./types";

export const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

export class ApiClientError extends Error {
  code: string;
  status: number;

  constructor(code: string, message: string, status: number) {
    super(message);
    this.name = "ApiClientError";
    this.code = code;
    this.status = status;
  }
}

function buildUrl(
  path: string,
  params?: Record<string, string | number | undefined>
): string {
  const url = new URL(path, API_BASE);
  if (params) {
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== "") {
        url.searchParams.set(key, String(value));
      }
    });
  }
  return url.toString();
}

async function parseResponse<T>(res: Response): Promise<ApiResponse<T>> {
  const body = (await res.json()) as ApiResponse<T>;

  if (!res.ok || body.error) {
    const err = body.error || { code: "unknown", message: res.statusText };
    throw new ApiClientError(err.code, err.message, res.status);
  }

  return body;
}

export async function apiGet<T>(
  path: string,
  params?: Record<string, string | number | undefined>
): Promise<ApiResponse<T>> {
  const res = await fetch(buildUrl(path, params), {
    headers: { Accept: "application/json" },
  });
  return parseResponse<T>(res);
}

export async function apiPost<T>(
  path: string,
  body?: unknown
): Promise<ApiResponse<T>> {
  const res = await fetch(buildUrl(path), {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  return parseResponse<T>(res);
}
