import type { GatewayClient } from "@/lib/api/client";

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
