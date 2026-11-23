import type { components } from "../../../../../common/api/schema";
import { bearerHeader, gatewayRequest } from "../../../../common/api/controller";

export type InventoryEntry = components["schemas"]["InventoryEntry"];

export function loadInventory(token: string, signal: AbortSignal) {
  return gatewayRequest("/inventory", {
    headers: bearerHeader(token), signal,
  }).then((response) => response.json() as Promise<InventoryEntry[]>);
}
