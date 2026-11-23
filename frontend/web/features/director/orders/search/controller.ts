import { useQuery } from "@tanstack/react-query";
import type { GatewayClient } from "@/lib/api/client";
import type { OrderFilters } from "./form";

export function useDirectorOrders(gateway: GatewayClient, filters: OrderFilters, enabled: boolean) {
  return useQuery({
    queryKey: ["director", "orders", filters],
    queryFn: async ({ signal }) => {
      const { data } = await gateway.GET("/director/orders", { params: { query: filters }, signal });
      if (!data) throw new Error("Empty director orders response");
      return data;
    },
    enabled,
  });
}
