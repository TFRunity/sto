import type { GatewayClient } from "@/lib/api/client";

export const customerVehicleListKey = ["customer", "vehicles"] as const;

export function vehicleListQuery(gateway: GatewayClient) {
  return {
    queryKey: customerVehicleListKey,
    queryFn: ({ signal }: { signal: AbortSignal }) => gateway.GET("/vehicles", { signal })
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
