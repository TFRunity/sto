import { createGatewayClient } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";
import { acceptSession } from "./browser-session";

type Schemas = components["schemas"];
const publicGateway = createGatewayClient();

export function signIn(body: Schemas["LoginRequest"], signal: AbortSignal) {
  return publicGateway.POST("/login", { body, signal })
    .then(({ data }) => acceptSession(data?.token, signal));
}

export function register(body: Schemas["RegisterRequest"], signal: AbortSignal) {
  return publicGateway.POST("/register", { body, signal })
    .then(({ data }) => acceptSession(data?.token, signal));
}

// Снимок токена в запросе защищает новую сессию от запоздалого 401 старой.
export function createSessionGateway(getToken: () => string | undefined, onUnauthorized: () => void) {
  const client = createGatewayClient();

  client.use({
    onRequest({ request }) {
      const token = getToken();
      if (token) request.headers.set("Authorization", `Bearer ${token}`);
    },
    onResponse({ request, response }) {
      const token = getToken();
      if (response.status === 401 && token && request.headers.get("Authorization") === `Bearer ${token}`) {
        onUnauthorized();
      }
    },
  });

  return client;
}
