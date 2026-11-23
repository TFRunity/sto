import type { GatewayClient } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";

export type HistoryItem = components["schemas"]["OrderHistoryItem"];

export const customerOrderHistoryKey = ["customer", "orders"] as const;

export function historyQuery(gateway: GatewayClient, vehicleId?: string) {
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

export function orderQuery(gateway: GatewayClient, id: string) {
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

export function createBooking(gateway: GatewayClient, body: components["schemas"]["CreateServiceBookingRequest"]) {
  return gateway.POST("/orders", { body }).then(({ data }) => {
    if (!data?.order_id) throw new Error("Missing order ID");
    return data.order_id;
  });
}
