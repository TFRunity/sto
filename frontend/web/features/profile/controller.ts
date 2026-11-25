import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query";
import type { GatewayClient } from "@/lib/api/client";

const profileKey = ["profile", "me"] as const;

export function profileQuery(gateway: GatewayClient) {
  return queryOptions({
    queryKey: profileKey,
    queryFn: async ({ signal }) => {
      const { data } = await gateway.GET("/me", { signal });
      if (!data) throw new Error("Profile is missing");
      return data;
    },
  });
}

export function useNotifications(gateway: GatewayClient) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (enabled: boolean) => {
      const { data } = await gateway.PATCH("/me", { body: { notifications_enabled: enabled } });
      if (!data) throw new Error("Updated profile is missing");
      return data;
    },
    onMutate: () => queryClient.cancelQueries({ queryKey: profileKey }),
    // Ответ старой сессии может прийти после выхода. Перечитываем профиль текущей сессии без записи старого ответа в кэш.
    onSettled: () => queryClient.invalidateQueries({ queryKey: profileKey }),
  });
}
