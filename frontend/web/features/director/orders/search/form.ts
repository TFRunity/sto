import type { paths } from "@shared/api/schema";

export type OrderFilters = paths["/director/orders"]["get"]["parameters"]["query"] & { page: number };
type OrderStatus = NonNullable<OrderFilters["status"]>;

export const statusOptions = {
  requested: "Заявка",
  active: "В работе",
  awaiting_payment: "Ожидает оплаты",
  paid: "Оплачен",
  cancelled: "Отменён",
} satisfies Record<OrderStatus, string>;

function readStatus(value: string | null): OrderFilters["status"] {
  if (value && Object.hasOwn(statusOptions, value)) return value as OrderStatus;
}

export function readFilters(params: Pick<URLSearchParams, "get">): OrderFilters {
  const page = Number(params.get("page"));
  return {
    q: params.get("q")?.trim() || undefined,
    status: readStatus(params.get("status")),
    page: Number.isInteger(page) && page >= 1 && page <= 2_147_483_647 ? page : 1,
  };
}

export function validateQuery(q?: string) {
  const length = q ? Array.from(q).length : 0;
  if (length === 1) return "Введите хотя бы два символа.";
  if (length > 200) return "Введите не больше 200 символов.";
}

export function ordersHref({ q, status, page }: OrderFilters) {
  const params = new URLSearchParams();
  if (q) params.set("q", q);
  if (status) params.set("status", status);
  params.set("page", String(page));
  return `/director/orders?${params}`;
}
