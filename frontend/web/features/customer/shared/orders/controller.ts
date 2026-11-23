import { useQuery } from "@tanstack/react-query";
import type { GatewayClient } from "@/lib/api/client";
import type { components } from "@shared/api/schema";
import { isPaymentPending } from "@shared/orders/payment";

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
    refetchInterval: ({ state }: { state: { data: HistoryItem[] | undefined } }) =>
      state.data?.some((order) => order.status === "paid" &&
        (isPaymentPending(order) || order.stage !== "выдан")) ? 5_000 : false,
  };
}

export function useHistory(gateway: GatewayClient, vehicleId?: string, enabled = true) {
  return useQuery({ ...historyQuery(gateway, vehicleId), enabled });
}
