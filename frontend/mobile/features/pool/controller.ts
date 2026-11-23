import type { components } from "../../../common/api/schema";
import { bearerHeader, gatewayRequest } from "../../common/api/controller";

export type Booking = components["schemas"]["ServiceBookingListItem"];

export function loadBookingPool(token: string, signal: AbortSignal) {
  return gatewayRequest("/orders?status=requested", {
    headers: bearerHeader(token), signal,
  }).then((response) => response.json() as Promise<Booking[]>);
}

export function claimBooking(token: string, id: string, signal: AbortSignal) {
  return gatewayRequest(`/orders/${encodeURIComponent(id)}/claim`, {
    method: "PATCH", headers: bearerHeader(token), signal,
  }).then(() => undefined);
}
