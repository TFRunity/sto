import { bearerHeader, gatewayRequest } from "../../../common/api/controller";
import type { IntakeClient, IntakeVehicle } from "../form";

export function searchClients(token: string, query: string, signal: AbortSignal) {
  return gatewayRequest(`/engineer/clients?q=${encodeURIComponent(query)}`, {
    headers: bearerHeader(token), signal,
  }).then((response) => response.json() as Promise<IntakeClient[]>);
}

export function findVehicle(token: string, number: string, signal: AbortSignal) {
  return gatewayRequest(`/engineer/vehicles?number=${encodeURIComponent(number)}`, {
    headers: bearerHeader(token), signal,
  }).then((response) => response.json() as Promise<IntakeVehicle>);
}
