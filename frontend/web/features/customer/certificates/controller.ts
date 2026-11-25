import { queryOptions } from "@tanstack/react-query";
import type { components } from "@shared/api/schema";
import type { GatewayClient } from "@/lib/api/client";

export type Certificate = components["schemas"]["ClientCertificate"];

export function certificatesQuery(gateway: GatewayClient) {
  return queryOptions({
    queryKey: ["customer", "certificates"],
    queryFn: async ({ signal }) => {
      const { data } = await gateway.GET("/certificates", { signal });
      if (!data) throw new Error("Certificate list is missing");
      return data;
    },
    refetchInterval: ({ state }) => state.data?.some((item) => item.status === "processing") ? 5_000 : false,
  });
}
