import { useQuery } from "@tanstack/react-query";
import type { GatewayClient } from "@/lib/api/client";
import { useHistory } from "../shared/orders/controller";
import { vehicleListQuery } from "../shared/vehicles/controller";

export function useCustomerOverview(gateway: GatewayClient, includeHistory: boolean) {
  const history = useHistory(gateway, undefined, includeHistory);
  const vehicleList = useQuery(vehicleListQuery(gateway));

  return { history, vehicleList };
}
