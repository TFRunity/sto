import type { components } from "../../../common/api/schema";
import { bearerHeader, gatewayRequest } from "../../common/api/controller";

export type Profile = components["schemas"]["MyProfile"];

export function loadProfile(token: string, signal: AbortSignal) {
  return gatewayRequest("/me", { headers: bearerHeader(token), signal })
    .then((response) => response.json() as Promise<Profile>);
}

export function updateNotifications(token: string, enabled: boolean, signal: AbortSignal) {
  return gatewayRequest("/me", {
    method: "PATCH", headers: { ...bearerHeader(token), "Content-Type": "application/json" },
    body: JSON.stringify({ notifications_enabled: enabled }), signal,
  }).then(() => undefined);
}
