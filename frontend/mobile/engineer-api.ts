import type { components } from "../web/lib/api/schema";

type LoginRequest = components["schemas"]["LoginRequest"];
type TokenResponse = components["schemas"]["TokenResponse"];
export type Booking = components["schemas"]["ServiceBookingListItem"];

const baseUrl = (process.env.EXPO_PUBLIC_API_URL ?? "http://10.0.2.2/api").replace(/\/+$/, "");

export class EngineerApiError extends Error {
  constructor(readonly status: number) {
    super(`Gateway returned HTTP ${status}`);
  }
}

export function loginEngineer(body: LoginRequest) {
  return fetch(`${baseUrl}/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  }).then((response) => {
    if (!response.ok) throw new EngineerApiError(response.status);
    return response.json() as Promise<TokenResponse>;
  }).then(({ token }) => {
    if (!token) throw new Error("Missing access token");
    return token;
  });
}

export function loadBookingPool(token: string, signal: AbortSignal) {
  return fetch(`${baseUrl}/orders?status=requested`, {
    headers: { Authorization: `Bearer ${token}` },
    signal,
  }).then((response) => {
    if (!response.ok) throw new EngineerApiError(response.status);
    return response.json() as Promise<Booking[]>;
  });
}
