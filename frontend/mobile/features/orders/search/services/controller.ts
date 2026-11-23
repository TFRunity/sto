import type { components } from "../../../../../common/api/schema";
import { bearerHeader, gatewayRequest } from "../../../../common/api/controller";

export type ServiceEntry = components["schemas"]["ServiceEntry"];

export function loadServices(token: string, signal: AbortSignal) {
  return gatewayRequest("/services", {
    headers: bearerHeader(token), signal,
  }).then((response) => response.json() as Promise<ServiceEntry[]>);
}
