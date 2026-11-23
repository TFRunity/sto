import { createGatewayClient } from "@/lib/api/client";

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
