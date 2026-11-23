import { useMutation, useQueryClient } from "@tanstack/react-query";
import { GatewayError, type GatewayClient } from "@/lib/api/client";
import type { components } from "@shared/api/schema";
import { isPaymentPending } from "@shared/orders/payment";
import { customerOrderHistoryKey } from "../shared/orders/controller";

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
      if (isPaymentPending(order)) return 1_000;
      return order.stage !== "выдан" && 5_000;
    },
  };
}

export function usePayOrder(gateway: GatewayClient, id: string) {
  const client = useQueryClient();
  const { queryKey } = orderQuery(gateway, id);

  return useMutation({
    mutationFn: () => gateway.POST("/orders/{id}/pay", { params: { path: { id } } }),
    onSuccess: async () => {
      await client.cancelQueries({ queryKey });
      void client.invalidateQueries({ queryKey: customerOrderHistoryKey });
      await client.invalidateQueries({ queryKey });
    },
    onError: async () => {
      await client.invalidateQueries({ queryKey });
    },
  });
}

export function useDownloadCertificate(gateway: GatewayClient, id: string) {
  return useMutation({
    mutationFn: async () => {
      try {
        const { data } = await gateway.GET("/orders/{id}/certificate", {
          params: { path: { id } },
          headers: { Accept: "application/pdf" },
          parseAs: "blob",
        });
        if (!data?.size) throw new Error("Certificate PDF is empty");
        return data;
      } catch (error) {
        if (error instanceof GatewayError && error.status === 404) return null;
        throw error;
      }
    },
  });
}
