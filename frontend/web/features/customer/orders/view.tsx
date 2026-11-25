"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { formatRubles } from "@shared/money/utils";
import { orderStageLabels } from "@shared/orders/labels";
import { isPaidAndStockDeducted } from "@shared/orders/payment";
import { useSession } from "@/common/session/view";
import type { components } from "@shared/api/schema";
import { ReadState, link, panel } from "../shared/view";

import { CertificateDownload } from "@/common/certificates/view";
import { vehiclePath } from "../shared/routes";
import { vehicleQuery } from "../shared/vehicles/controller";
import { vehicleLabel } from "../shared/utils";
import { OrderStatus } from "../shared/orders/view";
import { isOrderInaccessible, orderQuery, usePayOrder } from "./controller";

export function OrderScreen({ id }: { id: string }) {
  const { gateway } = useSession();
  const order = useQuery(orderQuery(gateway, id));
  const inaccessible = isOrderInaccessible(order.error);

  return (
    <div className="mx-auto max-w-2xl space-y-4">
      <Link href="/customer" className={link}>Мои заказы</Link>
      <ReadState pending={order.isPending} error={order.error} retry={() => void order.refetch()} />
      {order.data && !inaccessible && (
        <OrderCard key={id} order={order.data} canPay={!order.isError} refresh={() => void order.refetch()} />
      )}
    </div>
  );
}

function OrderCard({ order, canPay, refresh }: {
  order: components["schemas"]["Order"];
  canPay: boolean;
  refresh: () => void;
}) {
  const { gateway } = useSession();
  const { data: vehicle } = useQuery(vehicleQuery(gateway, order.vehicle_id));
  const payment = usePayOrder(gateway, order.id);

  return (
    <section className={`${panel} space-y-4`}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1>Детали заказа</h1>
        <OrderStatus order={order} />
      </div>
      <Link className={link} href={vehiclePath(order.vehicle_id)}>
        {vehicle ? vehicleLabel(vehicle) : "Карточка ТС"}
      </Link>
      {order.status !== "requested" && <p>Этап: {orderStageLabels[order.stage]}</p>}
      {(order.items.length > 0 || order.services.length > 0) && (
        <ul className="divide-y">
          {order.items.map((item) => (
            <li key={item.id} className="py-2">{item.name || "Товар"} × {item.quantity} · {formatRubles(item.price)} за шт.</li>
          ))}
          {order.services.map((service) => (
            <li key={service.id} className="py-2">{service.name} × {service.quantity} · {formatRubles(service.price)} за ед.</li>
          ))}
        </ul>
      )}
      <p className="font-semibold">Итого {formatRubles(order.total)}</p>
      {payment.isPending && order.status === "active" && (
        <p role="status">Сохраняем отметку об оплате…</p>
      )}
      {order.status === "active" && !payment.isPending && (
        <div className="space-y-2">
          {payment.isError && (
            <>
              <p role="alert">Не удалось подтвердить результат. Обновите заказ.</p>
              <Button variant="outline" onClick={refresh}>Обновить заказ</Button>
            </>
          )}
          <Button disabled={!canPay} onClick={() => payment.mutate()}>Отметить оплаченным</Button>
        </div>
      )}
      {isPaidAndStockDeducted(order) && <CertificateDownload id={order.id} />}
    </section>
  );
}
