const baseUrl = (process.env.EXPO_PUBLIC_API_URL ?? "http://10.0.2.2/api").replace(/\/+$/, "");

export class EngineerApiError extends Error {
  constructor(readonly status: number) {
    super(`Gateway returned HTTP ${status}`);
  }
}

export function gatewayRequest(path: string, init?: RequestInit) {
  return fetch(`${baseUrl}${path}`, init).then((response) => {
    if (!response.ok) throw new EngineerApiError(response.status);
    return response;
  });
}

export function bearerHeader(token: string) {
  return { Authorization: `Bearer ${token}` };
}

export function sessionErrorMessage(error: unknown) {
  if (!(error instanceof EngineerApiError)) return undefined;
  if (error.status === 401) return "Сессия истекла. Войдите снова.";
  if (error.status === 403) return "Вход доступен только инженеру.";
}
