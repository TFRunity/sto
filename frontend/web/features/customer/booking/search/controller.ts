import type { GatewayClient } from "@/lib/api/client";
import { createSearchMatcher } from "@shared/search/utils";

export function matchServiceNames(names: string[], input: string) {
  const matches = createSearchMatcher(input, "substring");
  return names.filter((name) => matches(name)).slice(0, 20);
}

export function bookingServiceNamesQuery(gateway: GatewayClient) {
  return {
    queryKey: ["customer", "booking-service-names"],
    staleTime: 5 * 60 * 1000,
    retry: false,
    queryFn: ({ signal }: { signal: AbortSignal }) => gateway.GET("/services", { signal })
      .then(({ data }) => [...new Set(data?.map(({ name }) => name))]),
  };
}
