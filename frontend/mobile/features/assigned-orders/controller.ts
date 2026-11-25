import type { paths } from "../../../common/api/schema";
import { bearerHeader, gatewayRequest } from "../../common/api/controller";
import type { HistoryOrder } from "../../common/order-history/view";

export type OrderScope = paths["/engineer/orders"]["get"]["parameters"]["query"]["scope"];

export function loadAssignedOrders(token: string, scope: OrderScope, signal: AbortSignal) {
  return gatewayRequest(`/engineer/orders?scope=${scope}`, {
    headers: bearerHeader(token), signal,
  }).then((response) => response.json() as Promise<HistoryOrder[]>);
}
