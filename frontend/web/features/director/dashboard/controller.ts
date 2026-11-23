import { useQuery } from "@tanstack/react-query";
import type { GatewayClient } from "@/lib/api/client";

export function useDirectorDashboard(gateway: GatewayClient) {
  return useQuery({
    queryKey: ["director", "dashboard"],
    queryFn: async ({ signal }) => {
      const { data } = await gateway.GET("/dashboard", { signal });
      if (!data) throw new Error("Empty dashboard response");
      return data;
    },
    retry: false,
  });
}
