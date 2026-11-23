import type { GatewayClient } from "@/lib/api/client";

export function bookingServiceNamesQuery(gateway: GatewayClient) {
  return {
    queryKey: ["customer", "booking-service-names"],
    staleTime: 5 * 60 * 1000,
    retry: false,
    queryFn: ({ signal }: { signal: AbortSignal }) => gateway.GET("/services", { signal })
      .then(({ data }) => [...new Set(data?.map(({ name }) => name))]),
  };
}
