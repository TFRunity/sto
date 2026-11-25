import { queryOptions } from "@tanstack/react-query";
import type { components } from "@shared/api/schema";
import { GatewayError, type GatewayClient } from "@/lib/api/client";

export type DirectorOrder = components["schemas"]["DirectorOrderDetails"];

export function directorOrderQuery(gateway: GatewayClient, id: string) {
  return queryOptions({
    queryKey: ["director", "order", id],
    queryFn: async ({ signal }) => {
      const { data } = await gateway.GET("/director/orders/{id}", { params: { path: { id } }, signal });
      if (!data) throw new Error("Order details are missing");
      return data;
    },
    refetchInterval: ({ state }) => {
      if (state.error instanceof GatewayError && [403, 404].includes(state.error.status)) return false;
      return state.data?.order.certificate_status === "processing" ? 5_000 : false;
    },
  });
}
