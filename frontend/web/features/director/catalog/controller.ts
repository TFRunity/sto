import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { components } from "@shared/api/schema";
import type { GatewayClient } from "@/lib/api/client";
import type { CatalogInput, CatalogKind } from "./form";

export type CatalogEntry = components["schemas"]["InventoryEntry" | "ServiceEntry"];

function catalogKey(kind: CatalogKind) {
  return ["director", "catalog", kind];
}

export function useCatalog(gateway: GatewayClient, kind: CatalogKind) {
  return useQuery({
    queryKey: catalogKey(kind),
    queryFn: async ({ signal }) => {
      const { data } = await gateway.GET(kind === "inventory" ? "/inventory" : "/services", { signal });
      if (!data) throw new Error("Empty catalog response");
      return data;
    },
  });
}

type CatalogChange =
  | { action: "save"; input: CatalogInput; id?: string }
  | { action: "delete"; kind: CatalogKind; id: string };

async function changeCatalog(gateway: GatewayClient, change: CatalogChange) {
  if (change.action === "delete") {
    await gateway.DELETE(change.kind === "inventory" ? "/inventory/{id}" : "/services/{id}", {
      params: { path: { id: change.id } },
    });
    return;
  }

  const { input, id } = change;
  if (input.kind === "inventory") {
    if (id) {
      await gateway.PUT("/inventory/{id}", { params: { path: { id } }, body: input.body });
    } else {
      await gateway.POST("/inventory", { body: input.body });
    }
  } else if (id) {
    await gateway.PUT("/services/{id}", { params: { path: { id } }, body: input.body });
  } else {
    await gateway.POST("/services", { body: input.body });
  }
}

export function useCatalogChange(gateway: GatewayClient) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (change: CatalogChange) => changeCatalog(gateway, change),
    onSuccess: (_, change) => queryClient.invalidateQueries({
      queryKey: catalogKey(change.action === "save" ? change.input.kind : change.kind),
    }),
  });
}
