import type { QueryClient } from "@tanstack/react-query";
import type { GatewayClient } from "@/lib/api/client";
import type { components } from "@shared/api/schema";
import { customerOrderHistoryKey } from "../shared/orders/controller";

export function createBooking(gateway: GatewayClient, queryClient: QueryClient, body: components["schemas"]["CreateServiceBookingRequest"]) {
  return gateway.POST("/orders", { body }).then(({ data }) => {
    if (!data?.order_id) throw new Error("Missing order ID");
    void queryClient.invalidateQueries({ queryKey: customerOrderHistoryKey });
    return data.order_id;
  });
}
