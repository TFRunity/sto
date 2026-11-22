import type { useSession } from "@/features/auth/session-provider";
import type { components } from "@/lib/api/schema";

export type CustomerGateway = ReturnType<typeof useSession>["gateway"];
export type HistoryItem = components["schemas"]["OrderHistoryItem"];
export type VehicleMake = components["schemas"]["VehicleMake"];

export const customerOrderHistoryKey = ["customer", "orders"] as const;
export const customerVehicleListKey = ["customer", "vehicles"] as const;

export function vehicleListQuery(gateway: CustomerGateway) {
  return {
    queryKey: customerVehicleListKey,
    queryFn: ({ signal }: { signal: AbortSignal }) => gateway.GET("/vehicles", { signal })
      .then(({ data }) => data ?? []),
  };
}

export function vehicleMakesQuery(gateway: CustomerGateway) {
  return {
    queryKey: ["customer", "vehicle-makes"],
    staleTime: 60 * 60 * 1000,
    queryFn: ({ signal }: { signal: AbortSignal }) => gateway.GET("/vehicles/makes", { signal })
      .then(({ data }) => data ?? []),
  };
}

export function vehicleQuery(gateway: CustomerGateway, id: string) {
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

export function historyQuery(gateway: CustomerGateway, vehicleId?: string) {
  return {
    queryKey: [...customerOrderHistoryKey, vehicleId ?? "all"],
    queryFn: ({ signal }: { signal: AbortSignal }) => gateway.GET("/orders", {
      params: { query: { client_id: "me", vehicle_id: vehicleId } },
      signal,
    }).then(({ data }) => {
      // Этот маршрут также отдаёт пул инженера; запись в истории имеет этап работ.
      return (data ?? []).filter((entry): entry is HistoryItem => "stage" in entry);
    }),
  };
}

export function orderQuery(gateway: CustomerGateway, id: string) {
  return {
    queryKey: ["customer", "order", id],
    queryFn: ({ signal }: { signal: AbortSignal }) => gateway.GET("/orders/{id}", {
      params: { path: { id } },
      signal,
    }).then(({ data }) => {
      if (!data) throw new Error("Order is missing");
      return data;
    }),
  };
}

export function vehiclePhotoQuery(gateway: CustomerGateway, id: string, photoId?: string) {
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

export function createVehicle(gateway: CustomerGateway, body: components["schemas"]["VehicleDetails"]) {
  return gateway.POST("/vehicles", { body }).then(({ data }) => {
    if (!data?.vehicle_id) throw new Error("Missing vehicle ID");
    return data.vehicle_id;
  });
}

export function createBooking(gateway: CustomerGateway, body: components["schemas"]["CreateServiceBookingRequest"]) {
  return gateway.POST("/orders", { body }).then(({ data }) => {
    if (!data?.order_id) throw new Error("Missing order ID");
    return data.order_id;
  });
}

export function replaceVehicleCover(gateway: CustomerGateway, id: string, photo: File) {
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
