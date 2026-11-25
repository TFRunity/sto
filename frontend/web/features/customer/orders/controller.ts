import { useMutation, useQueryClient } from "@tanstack/react-query";
import { GatewayError, type GatewayClient } from "@/lib/api/client";
import type { components } from "@shared/api/schema";
import { customerOrderHistoryKey, type HistoryItem } from "../shared/orders/controller";

type Order = components["schemas"]["Order"];

export function isOrderInaccessible(error: unknown) {
  return error instanceof GatewayError && (error.status === 403 || error.status === 404);
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
    refetchInterval: ({ state }: { state: { data: Order | undefined; error: Error | null } }) => {
      if (isOrderInaccessible(state.error)) return false;
      const order = state.data;
      if (!order || order.status !== "paid") return false;
      return order.stage !== "выдан" && 5_000;
    },
  };
}

export function usePayOrder(gateway: GatewayClient, id: string) {
  const client = useQueryClient();
  const { queryKey } = orderQuery(gateway, id);

  return useMutation({
    mutationFn: () => gateway.POST("/orders/{id}/pay", { params: { path: { id } } }),
    onSuccess: async ({ data }) => {
      if (!data) return;
      const history = { queryKey: customerOrderHistoryKey };

      // Останавливаем чтения, начатые до отметки, чтобы они не вернули старый статус в кэш.
      await Promise.all([client.cancelQueries({ queryKey }), client.cancelQueries(history)]);
      client.setQueryData<Order>(queryKey, (order) => order && {
        ...order,
        status: data.status,
        payment_confirmed: true,
        // Ответ POST подтверждает только оплату. Готовность справки определит чтение заказа.
        stock_status: order.stock_status ?? "pending",
      });
      client.setQueriesData<HistoryItem[]>(history, (orders) => orders?.map((order) => order.id === id
        ? { ...order, status: data.status, payment_confirmed: true }
        : order));
    },
    onSettled: async () => {
      void client.invalidateQueries({ queryKey: customerOrderHistoryKey });
      await client.invalidateQueries({ queryKey });
    },
  });
}
