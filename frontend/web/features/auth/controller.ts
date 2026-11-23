import { createGatewayClient } from "@/lib/api/client";
import type { components } from "@shared/api/schema";
import { acceptSession } from "@/common/session/session";

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
