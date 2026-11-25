import type { components } from "../api/schema";

type PaymentFields = Pick<components["schemas"]["Order"], "status" | "payment_confirmed">;

export function isPaymentPending(order: PaymentFields) {
  return order.status === "paid" && !order.payment_confirmed;
}

export function isPaymentConfirmed(order: PaymentFields) {
  return order.status === "paid" && order.payment_confirmed;
}

export function isPaidAndStockDeducted(order: PaymentFields & Pick<components["schemas"]["Order"], "stock_status">) {
  return isPaymentConfirmed(order) && (order.stock_status === undefined || order.stock_status === "deducted");
}
