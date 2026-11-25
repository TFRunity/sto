import type { components } from "../../../common/api/schema";
import { bearerHeader, gatewayRequest } from "../../common/api/controller";

export type Vehicle = components["schemas"]["VehicleCard"];

export function loadVehicle(token: string, vehicleId: string, signal: AbortSignal) {
  return gatewayRequest(`/vehicles/${encodeURIComponent(vehicleId)}`, {
    headers: bearerHeader(token), signal,
  }).then((response) => response.json() as Promise<Vehicle>);
}

export function loadVehicleHistory(token: string, orderId: string, signal: AbortSignal) {
  return gatewayRequest(`/orders/${encodeURIComponent(orderId)}/vehicle-history`, {
    headers: bearerHeader(token), signal,
  }).then((response) => response.json() as Promise<components["schemas"]["OrderHistoryItem"][]>);
}
