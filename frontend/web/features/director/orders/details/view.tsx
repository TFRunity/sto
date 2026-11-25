"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { formatRubles } from "@shared/money/utils";
import { certificateStatusLabels } from "@shared/orders/certificate";
import { orderStageLabels, orderStatusLabels } from "@shared/orders/labels";
import { CertificateDownload } from "@/common/certificates/view";
import { useSession } from "@/common/session/view";
import { Button } from "@/components/ui/button";
import { GatewayError } from "@/lib/api/client";
import { directorOrderQuery, type DirectorOrder } from "./controller";

const section = "space-y-2 rounded-xl border bg-card p-4 sm:p-5";
const textLink = "font-medium underline-offset-4 hover:underline";

export function DirectorOrderScreen({ id }: { id: string }) {
  const { gateway } = useSession();
  const details = useQuery(directorOrderQuery(gateway, id));
  const unavailable = details.error instanceof GatewayError && [403, 404].includes(details.error.status);

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <Link href="/director/orders" className={textLink}>Все заказы</Link>
      <h1>Заказ {id.slice(0, 8)}</h1>
      {details.isPending && <p role="status">Загружаем заказ…</p>}
      {details.isError && (
        <div className="space-y-2">
          <p role="alert">{unavailable ? "Заказ недоступен." : "Не удалось загрузить заказ."}</p>
          {!unavailable && <Button variant="outline" onClick={() => void details.refetch()}>Повторить</Button>}
        </div>
      )}
      {details.data && !unavailable && <OrderDetails details={details.data} />}
    </div>
  );
}

function OrderDetails({ details: { order, client, vehicle } }: { details: DirectorOrder }) {
  return (
    <>
      <section className={section} aria-label="Статус заказа">
        <p>{orderStatusLabels[order.status]} · {orderStageLabels[order.stage]}</p>
        <p className="font-semibold">Итого {formatRubles(order.total)}</p>
      </section>
      <div className="grid gap-4 sm:grid-cols-2">
        <section className={section}>
          <h2>Клиент</h2>
          {client ? <>
            <p>{client.name}</p>
            <a href={`tel:${client.phone}`} className={textLink}>{client.phone}</a>
            <p><Link href={relatedOrdersHref(client.phone)} className={textLink}>Заказы клиента</Link></p>
          </> : <p>Данные клиента недоступны.</p>}
        </section>
        <section className={section}>
          <h2>Автомобиль</h2>
          {vehicle ? <>
            <p>{vehicle.brand ? `${vehicle.brand} · ` : ""}{vehicle.number}</p>
            {vehicle.vin && <p>VIN {vehicle.vin}</p>}
            <Link href={relatedOrdersHref(vehicle.number)} className={textLink}>Заказы по автомобилю</Link>
          </> : <p>Данные автомобиля недоступны.</p>}
        </section>
      </div>
      <section className={section}>
        <h2>Исполнители</h2>
        {order.engineers?.length ? (
          <ul className="space-y-1">
            {order.engineers.map((engineer) => (
              <li key={engineer.id}>
                {engineer.name || engineer.id}{engineer.id === order.engineer_id ? " · Ответственный" : ""}
              </li>
            ))}
          </ul>
        ) : <p>Исполнители ещё не назначены.</p>}
      </section>
      <OrderPositions order={order} />
      <section className={section}>
        <h2>Справка</h2>
        <p>{order.certificate_status ? certificateStatusLabels[order.certificate_status] : "Пока нет справки"}</p>
        {order.certificate_status === "ready" && <CertificateDownload id={order.id} />}
      </section>
    </>
  );
}

function OrderPositions({ order }: { order: DirectorOrder["order"] }) {
  const positions = [
    ...order.items.map((item) => ({ ...item, key: `item-${item.id}`, name: item.name || "Товар" })),
    ...order.services.map((service) => ({ ...service, key: `service-${service.id}` })),
  ];
  return (
    <section className={section}>
      <h2>Товары и услуги</h2>
      {positions.length ? (
        <ul className="divide-y">
          {positions.map((position) => (
            <li key={position.key} className="flex flex-wrap justify-between gap-x-3 py-2">
              <span>{position.name} × {position.quantity}</span>
              <span>{formatRubles(position.price)} за ед.</span>
            </li>
          ))}
        </ul>
      ) : <p>Позиций пока нет.</p>}
    </section>
  );
}

function relatedOrdersHref(query: string) {
  const params = new URLSearchParams({ q: query });
  return `/director/orders?${params}`;
}
