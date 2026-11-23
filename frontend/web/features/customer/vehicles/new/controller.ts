import type { QueryClient } from "@tanstack/react-query";
import type { GatewayClient } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";
import { customerVehicleListKey } from "../../shared/vehicles/controller";

export type VehicleMake = components["schemas"]["VehicleMake"];

export function vehicleMakesQuery(gateway: GatewayClient) {
  return {
    queryKey: ["customer", "vehicle-makes"],
    staleTime: 60 * 60 * 1000,
    queryFn: ({ signal }: { signal: AbortSignal }) => gateway.GET("/vehicles/makes", { signal })
      .then(({ data }) => data ?? []),
  };
}

export function createVehicle(gateway: GatewayClient, queryClient: QueryClient, body: components["schemas"]["VehicleDetails"]) {
  return gateway.POST("/vehicles", { body }).then(({ data }) => {
    if (!data?.vehicle_id) throw new Error("Missing vehicle ID");
    void queryClient.invalidateQueries({ queryKey: customerVehicleListKey });
    return data.vehicle_id;
  });
}
