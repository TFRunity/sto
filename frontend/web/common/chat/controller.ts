import { queryOptions, useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { subscribeToChat } from "@shared/chat/events";
import type { GatewayClient } from "@/lib/api/client";

const chatKey = ["chat"] as const;
const dialogsKey = [...chatKey, "dialogs"] as const;
const messagesKey = (id: string) => [...chatKey, "messages", id] as const;

export type ChatTarget =
  | { kind: "order"; id: string }
  | { kind: "director_engineer"; id: string }
  | { kind: "support" };

export function chatDialogsQuery(gateway: GatewayClient) {
  return queryOptions({
    queryKey: dialogsKey,
    queryFn: async ({ signal }) => (await gateway.GET("/chat/dialogs", { signal })).data ?? [],
  });
}

export function chatMessagesQuery(gateway: GatewayClient, id: string) {
  return queryOptions({
    queryKey: messagesKey(id),
    queryFn: async ({ signal }) => (await gateway.GET("/chat/dialogs/{id}/messages", {
      params: { path: { id } }, signal,
    })).data ?? [],
  });
}

export function useChatEvents(token: string | undefined) {
  const cache = useQueryClient();
  useEffect(() => {
    if (!token) return;
    return subscribeToChat(
      `${window.location.origin}/api`,
      token,
      () => void cache.invalidateQueries({ queryKey: chatKey }),
      ({ dialog_id }) => {
        void cache.invalidateQueries({ queryKey: dialogsKey });
        void cache.invalidateQueries({ queryKey: messagesKey(dialog_id) });
      },
    );
  }, [cache, token]);
}

export function useOpenChat(gateway: GatewayClient) {
  const cache = useQueryClient();
  return useMutation({
    mutationFn: async (target: ChatTarget) => {
      switch (target.kind) {
        case "order":
          return (await gateway.POST("/chat/dialogs/order", { body: { order_id: target.id } })).data;
        case "director_engineer":
          return (await gateway.POST("/chat/dialogs/director-engineer", { body: { engineer_id: target.id } })).data;
        case "support":
          return (await gateway.POST("/chat/dialogs/support")).data;
      }
    },
    onSuccess: () => void cache.invalidateQueries({ queryKey: dialogsKey }),
  });
}

export function useChatActions(gateway: GatewayClient, id: string) {
  const cache = useQueryClient();
  function refresh() {
    void cache.invalidateQueries({ queryKey: dialogsKey });
    void cache.invalidateQueries({ queryKey: messagesKey(id) });
  }
  const send = useMutation({
    mutationFn: (body: string) => gateway.POST("/chat/dialogs/{id}/messages", {
      params: { path: { id } }, body: { body },
    }),
    onSettled: refresh,
  });
  const markRead = useMutation({
    mutationFn: (throughId: string) => gateway.PATCH("/chat/dialogs/{id}/read", {
      params: { path: { id } }, body: { through_message_id: throughId },
    }),
    onSuccess: refresh,
  });
  return { send, markRead };
}
