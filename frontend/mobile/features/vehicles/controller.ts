import type { components } from "../../../common/api/schema";
import { bearerHeader, gatewayRequest } from "../../common/api/controller";

export type Vehicle = components["schemas"]["VehicleCard"];

export function loadVehicle(token: string, vehicleId: string, signal: AbortSignal) {
  return gatewayRequest(`/vehicles/${encodeURIComponent(vehicleId)}`, {
    headers: bearerHeader(token), signal,
  }).then((response) => response.json() as Promise<Vehicle>);
}
