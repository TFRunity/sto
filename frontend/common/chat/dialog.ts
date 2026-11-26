import type { components } from "../api/schema";

export type ChatDialog = components["schemas"]["ChatDialog"];
export type ChatMessage = components["schemas"]["ChatMessage"];

export function dialogTitle(dialog: ChatDialog, role: "client" | "director" | "engineer") {
  switch (dialog.kind) {
    case "order": {
      const order = dialog.order_id ? `Заказ ${dialog.order_id.slice(0, 8)}` : "Диалог по заказу";
      return dialog.can_write ? order : `${order} · Архив`;
    }
    case "support":
      return role === "director" ? `Поддержка · клиент ${dialog.peer_id.slice(0, 8)}` : "Поддержка";
    case "director_engineer":
      return `${role === "director" ? "Инженер" : "Директор"} ${dialog.peer_id.slice(0, 8)}`;
  }
}

export function lastUnreadMessage(messages: ChatMessage[], peerId: string) {
  return messages.findLast((message) => message.sender_id === peerId && !message.is_read);
}

export function canSendMessage(body: string) {
  return !!body.trim() && Array.from(body).length <= 4000 && !body.includes("\0");
}
