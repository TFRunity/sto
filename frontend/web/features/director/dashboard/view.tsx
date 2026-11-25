"use client";

import Link from "next/link";
import { formatRubles } from "@shared/money/utils";
import { orderStatusLabels } from "@shared/orders/labels";
import { useSession } from "@/common/session/view";
import { Button } from "@/components/ui/button";
import { useDirectorDashboard } from "./controller";

const statusLabels: Readonly<Partial<Record<string, string>>> = {
  ...orderStatusLabels,
  awaiting_payment: "Ожидает оплаты",
};
const panel = "rounded-lg border bg-card p-4";

function recentOrderLabel(status: string, stage: string) {
  if (stage === "выдан") return "Выдан";
  if (status === "active" && stage === "готов") return statusLabels.awaiting_payment;
  return statusLabels[status] ?? status;
}

export function DirectorDashboard() {
  const { gateway } = useSession();
  const dashboard = useDirectorDashboard(gateway);

  if (dashboard.isPending) return <p role="status">Загружаем дашборд…</p>;
  if (dashboard.isError) {
    return (
      <div className={`${panel} space-y-3 text-center`}>
        <p role="alert">Не удалось загрузить дашборд</p>
        <Button variant="outline" disabled={dashboard.isFetching} onClick={() => void dashboard.refetch()}>
          {dashboard.isFetching ? "Загружаем…" : "Повторить"}
        </Button>
      </div>
    );
  }

  const { orders, engineers } = dashboard.data;
  const metrics = [
    { label: "Выручка сегодня", value: formatRubles(orders.today_revenue) },
    { label: "Заказов сегодня", value: orders.today_orders },
    { label: "В работе", value: orders.active_orders },
    { label: "Ожидают оплаты", value: orders.awaiting_payment },
  ];

  return (
    <div className="space-y-5">
      <h1>Дашборд</h1>
      <dl className="grid grid-cols-2 gap-3 md:grid-cols-4">
        {metrics.map(({ label, value }) => (
          <div key={label} className={panel}>
            <dt className="text-sm text-muted-foreground">{label}</dt>
            <dd className="mt-1 text-xl font-semibold tabular-nums">{value}</dd>
          </div>
        ))}
      </dl>
      <div className="grid gap-4 md:grid-cols-[1.4fr_1fr]">
        <section className={panel} aria-labelledby="recent-orders-title">
          <h2 id="recent-orders-title" className="font-semibold">Последние заказы</h2>
          {orders.recent_orders.length ? (
            <ul className="mt-2 divide-y">
              {orders.recent_orders.map((order) => (
                <li key={order.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
                  <Link href={`/director/orders/${order.id}`} className="font-medium underline-offset-4 hover:underline">
                    {order.vehicle_number || `Заказ ${order.id.slice(0, 8)}`}
                  </Link>
                  <span className="rounded-md bg-muted px-2 py-1 text-sm">
                    {recentOrderLabel(order.status, order.stage)}
                  </span>
                </li>
              ))}
            </ul>
          ) : <p className="mt-2 text-sm text-muted-foreground">Заказов пока нет.</p>}
        </section>
        <section className={panel} aria-labelledby="engineer-load-title">
          <h2 id="engineer-load-title" className="font-semibold">Загрузка инженеров</h2>
          {engineers.length ? (
            <dl className="mt-2 divide-y">
              {engineers.map((engineer) => (
                <div key={engineer.id} className="flex items-baseline justify-between gap-3 py-2">
                  <dt>{engineer.name}</dt>
                  <dd className="text-sm tabular-nums text-muted-foreground">
                    {engineer.active_orders} в работе
                  </dd>
                </div>
              ))}
            </dl>
          ) : <p className="mt-2 text-sm text-muted-foreground">Инженеров пока нет.</p>}
        </section>
      </div>
    </div>
  );
}
