export type VehicleLabelData = { brand: string; number: string };

export function vehicleLabel({ brand, number }: VehicleLabelData) {
  return `${brand} · ${number}`;
}
