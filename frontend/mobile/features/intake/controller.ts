import type { components } from "../../../common/api/schema";
import { bearerHeader, gatewayRequest } from "../../common/api/controller";
import type { IntakeVehicle, IntakeVehicleRequest } from "./form";

export function createIntakeVehicle(token: string, vehicle: IntakeVehicleRequest, signal: AbortSignal) {
  return post(token, "/engineer/vehicles", vehicle, signal)
    .then((response) => response.json() as Promise<IntakeVehicle>);
}

export function createOrder(token: string, body: components["schemas"]["CreateEngineerOrderRequest"], signal: AbortSignal) {
  return post(token, "/orders", body, signal)
    .then((response) => response.json() as Promise<components["schemas"]["CreateOrderResponse"]>);
}

function post(token: string, path: string, body: object, signal: AbortSignal) {
  return gatewayRequest(path, {
    method: "POST",
    headers: { ...bearerHeader(token), "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal,
  });
}
