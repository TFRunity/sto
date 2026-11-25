import type { components } from "../../../common/api/schema";

export type IntakeVehicle = components["schemas"]["IntakeVehicle"];
export type IntakeVehicleRequest = components["schemas"]["CreateIntakeVehicleRequest"];
export type IntakeClient = components["schemas"]["PersonSummary"];

export function prepareVehicle(ownerId: string, number: string, brand: string, vin: string): IntakeVehicleRequest {
  return { owner_id: ownerId, number: number.trim(), brand: brand.trim(), vin: vin.trim() };
}
