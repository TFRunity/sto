import type { GatewayClient } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";

export type VehicleMake = components["schemas"]["VehicleMake"];

export const customerVehicleListKey = ["customer", "vehicles"] as const;

export function vehicleListQuery(gateway: GatewayClient) {
  return {
    queryKey: customerVehicleListKey,
    queryFn: ({ signal }: { signal: AbortSignal }) => gateway.GET("/vehicles", { signal })
      .then(({ data }) => data ?? []),
  };
}

export function vehicleMakesQuery(gateway: GatewayClient) {
  return {
    queryKey: ["customer", "vehicle-makes"],
    staleTime: 60 * 60 * 1000,
    queryFn: ({ signal }: { signal: AbortSignal }) => gateway.GET("/vehicles/makes", { signal })
      .then(({ data }) => data ?? []),
  };
}

export function vehicleQuery(gateway: GatewayClient, id: string) {
  return {
    queryKey: ["customer", "vehicle", id],
    queryFn: ({ signal }: { signal: AbortSignal }) => gateway.GET("/vehicles/{id}", {
      params: { path: { id } },
      signal,
    }).then(({ data }) => {
      if (!data) throw new Error("Vehicle is missing");
      return data;
    }),
  };
}

export function vehiclePhotoQuery(gateway: GatewayClient, id: string, photoId?: string) {
  return {
    queryKey: ["customer", "vehicle", id, "photo", photoId ?? "cover"],
    queryFn: ({ signal }: { signal: AbortSignal }) => {
      const response = photoId
        ? gateway.GET("/vehicles/{id}/photos/{photo_id}", {
            params: { path: { id, photo_id: photoId } }, parseAs: "blob", signal,
          })
        : gateway.GET("/vehicles/{id}/cover_photo", {
            params: { path: { id } }, parseAs: "blob", cache: "no-store", signal,
          });

      return response.then(({ data }) => {
        if (!data) throw new Error("Photo is missing");
        return data;
      });
    },
  };
}

export function createVehicle(gateway: GatewayClient, body: components["schemas"]["VehicleDetails"]) {
  return gateway.POST("/vehicles", { body }).then(({ data }) => {
    if (!data?.vehicle_id) throw new Error("Missing vehicle ID");
    return data.vehicle_id;
  });
}

export function replaceVehicleCover(gateway: GatewayClient, id: string, photo: File) {
  return gateway.PUT("/vehicles/{id}/cover_photo", {
    params: { path: { id } },
    body: { photo: photo.name },
    bodySerializer: () => {
      // OpenAPI описывает бинарное поле как string. В multipart передаём сам файл.
      const form = new FormData();
      form.set("photo", photo);
      return form;
    },
  });
}
