import type { Order, OrderWrite } from "./controller";
import type { PositionKind } from "./search/form";

export function availableStages(order: Order): Order["stage"][] {
  if (order.status !== "active" && order.status !== "paid") return [];
  const workStages: Order["stage"][] = ["принят", "в_работе", "готов"];
  return order.status === "paid" && order.payment_confirmed ? [...workStages, "выдан"] : workStages;
}

export function canCancelOrder(order: Order) {
  return order.status === "requested" || order.status === "active";
}

function readPositionQuantity(input: string) {
  const text = input.trim();
  if (!/^\d+$/.test(text)) return undefined;
  const quantity = Number(text);
  return quantity >= 1 && quantity <= 2147483647 ? quantity : undefined;
}

export function readPositionForm(kind: PositionKind, id: string, input: string): OrderWrite | undefined {
  const quantity = readPositionQuantity(input);
  if (quantity === undefined) return undefined;

  return kind === "item"
    ? { action: "items", body: { item_id: id, quantity } }
    : { action: "services", body: { service_id: id, quantity } };
}
