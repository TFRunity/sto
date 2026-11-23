import type { components } from "../../../common/api/schema";
import { bearerHeader, gatewayRequest } from "../../common/api/controller";

export type Order = components["schemas"]["Order"];

export function loadOrder(token: string, id: string, signal: AbortSignal) {
  return gatewayRequest(`/orders/${encodeURIComponent(id)}`, {
    headers: bearerHeader(token), signal,
  }).then((response) => response.json() as Promise<Order>);
}

export type OrderWrite =
  | { action: "items"; body: components["schemas"]["AddOrderItemRequest"] }
  | { action: "services"; body: components["schemas"]["AddOrderServiceRequest"] }
  | { action: "stage"; body: components["schemas"]["ChangeOrderStageRequest"] }
  | { action: "cancel" };

export function updateOrder(token: string, id: string, write: OrderWrite, signal: AbortSignal) {
  const isPosition = write.action === "items" || write.action === "services";
  const body = "body" in write ? JSON.stringify(write.body) : undefined;

  return gatewayRequest(`/orders/${encodeURIComponent(id)}/${write.action}`, {
    method: isPosition ? "POST" : "PATCH",
    headers: body ? { ...bearerHeader(token), "Content-Type": "application/json" } : bearerHeader(token),
    body,
    signal,
  }).then(() => undefined);
}
