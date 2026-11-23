"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useSession } from "@/common/session/view";
import type { components } from "@/lib/api/schema";
import { ReadState, link, panel } from "../shared/view";

import { vehiclePath } from "../shared/routes";
import { vehicleQuery } from "../shared/vehicles/controller";
import { vehicleLabel } from "../shared/utils";
import { OrderStatus, stageLabels } from "../shared/orders/view";
import { orderQuery } from "./controller";

const money = new Intl.NumberFormat("ru-RU", { style: "currency", currency: "RUB" });

export function OrderScreen({ id }: { id: string }) {
  const { gateway } = useSession();
  const order = useQuery(orderQuery(gateway, id));

  return (
    <div className="mx-auto max-w-2xl space-y-4">
      <Link href="/customer" className={link}>Мои заказы</Link>
      <ReadState pending={order.isPending} error={order.error} retry={() => void order.refetch()}>
        {order.data && <OrderCard order={order.data} />}
      </ReadState>
    </div>
  );
}

function OrderCard({ order }: { order: components["schemas"]["Order"] }) {
  const { gateway } = useSession();
  const { data: vehicle } = useQuery(vehicleQuery(gateway, order.vehicle_id));

  return (
    <section className={`${panel} space-y-4`}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1>Детали заказа</h1>
        <OrderStatus status={order.status} />
      </div>
      <Link className={link} href={vehiclePath(order.vehicle_id)}>
        {vehicle ? vehicleLabel(vehicle) : "Карточка ТС"}
      </Link>
      {order.status !== "requested" && <p>Этап: {stageLabels[order.stage]}</p>}
      {(order.items.length > 0 || order.services.length > 0) && (
        <ul className="divide-y">
          {order.items.map((item) => (
            <li key={item.id} className="py-2">Товар × {item.quantity} · {money.format(Number(item.price))} за шт.</li>
          ))}
          {order.services.map((service) => (
            <li key={service.id} className="py-2">{service.name} × {service.quantity} · {money.format(Number(service.price))} за ед.</li>
          ))}
        </ul>
      )}
      <p className="font-semibold">Итого {money.format(Number(order.total))}</p>
    </section>
  );
}
