"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { useSession } from "@/features/auth/session-provider";
import { GatewayError } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";
import { historyQuery, orderQuery, vehicleListQuery, vehicleQuery, type HistoryItem } from "./customer-api";
import { BookingForm } from "./forms";
import { VehiclePhotos } from "./vehicle-photos";

type Vehicle = components["schemas"]["VehicleCard"];

const statusLabels = { requested: "Заявка", active: "В работе", paid: "Оплачен", cancelled: "Отменён" };
const stageLabels = { принят: "Принят", в_работе: "В работе", готов: "Готов", выдан: "Выдан" };
const statusColors = {
  requested: "bg-accent text-accent-foreground",
  active: "bg-[#faeeda] text-[#854f0b]",
  paid: "bg-[#eaf3de] text-[#3b6d11]",
  cancelled: "bg-[#fcebeb] text-destructive",
};

const panel = "rounded-xl border bg-card p-4 sm:p-5";
const link = "font-medium text-accent-foreground underline-offset-4 hover:underline";
const vehiclePath = (id: string) => `/customer/vehicles/${id}`;
const orderPath = (id: string) => `/customer/orders/${id}`;
const dateFormat = new Intl.DateTimeFormat("ru-RU", { dateStyle: "medium" });
const money = new Intl.NumberFormat("ru-RU", { style: "currency", currency: "RUB" });

function useHistory(vehicleId?: string, enabled = true) {
  const { gateway } = useSession();
  return useQuery({ ...historyQuery(gateway, vehicleId), enabled });
}

function ReadState({ pending, error, retry, children }: {
  pending: boolean; error: Error | null; retry: () => void; children?: ReactNode;
}) {
  if (pending) return <p role="status" className="text-muted-foreground">Загружаем…</p>;
  if (!error) return children;
  if (error instanceof GatewayError && [403, 404].includes(error.status)) {
    return <p role="alert">Объект недоступен или удалён.</p>;
  }
  return (
    <div className="space-y-2">
      <p role="alert">Не удалось загрузить данные.</p>
      <Button variant="outline" onClick={retry}>Повторить</Button>
    </div>
  );
}

function Status({ status }: { status: HistoryItem["status"] }) {
  return <span className={`rounded-md px-2.5 py-1 text-sm ${statusColors[status]}`}>{statusLabels[status]}</span>;
}

function OrderList({ orders, vehicles }: { orders: HistoryItem[]; vehicles?: ReadonlyMap<string, Pick<Vehicle, "brand" | "number">> }) {
  return (
    <ul className="space-y-3">
      {orders.map((order) => {
        const vehicle = vehicles?.get(order.vehicle_id);
        return (
          <li key={order.id} className={`${panel} flex flex-wrap items-start justify-between gap-3`}>
            <div className="space-y-1">
              <Link href={orderPath(order.id)} className={link}>
                {vehicle ? `${vehicle.brand} · ${vehicle.number}` : `Заказ ${order.id.slice(0, 8)}`}
              </Link>
              <p className="text-sm text-muted-foreground">
                {dateFormat.format(new Date(order.created_at))}
                {order.status === "requested" ? "" : ` · ${stageLabels[order.stage]}`}
              </p>
              <Link href={vehiclePath(order.vehicle_id)} className={`text-sm ${link}`}>Карточка ТС</Link>
            </div>
            <Status status={order.status} />
          </li>
        );
      })}
    </ul>
  );
}

export function CustomerHistory({ choosing = false }: { choosing?: boolean }) {
  const { gateway } = useSession();
  const history = useHistory(undefined, !choosing);
  const vehicleList = useQuery(vehicleListQuery(gateway));
  const vehicles = new Map((vehicleList.data ?? []).map((vehicle) => [vehicle.vehicle_id, vehicle]));

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1>{choosing ? "Выберите автомобиль" : "Мои заказы"}</h1>
        {!choosing && <Button asChild><Link href="/customer/booking">Записаться на сервис</Link></Button>}
      </div>
      {!choosing && (
        <ReadState pending={history.isPending} error={history.error} retry={() => void history.refetch()}>
          {history.data?.length
            ? <OrderList orders={history.data} vehicles={vehicles} />
            : <p className={panel}>Заказов пока нет. Выберите автомобиль, чтобы записаться на сервис.</p>}
        </ReadState>
      )}

      <section aria-label="Автомобили" className="space-y-3">
        {!choosing && <h2 className="text-lg font-semibold">Мои ТС</h2>}
        <ReadState pending={vehicleList.isPending} error={vehicleList.error} retry={() => void vehicleList.refetch()} />
        <div className="grid gap-3 sm:grid-cols-2">
          {(vehicleList.data ?? []).map((vehicle) => {
            const path = vehiclePath(vehicle.vehicle_id);
            return (
              <Link key={vehicle.vehicle_id} href={choosing ? `${path}/booking` : path} className={`${panel} hover:border-ring`}>
                <span className="block font-medium">{vehicle.brand}</span>
                <span className="text-sm text-muted-foreground">{vehicle.number}</span>
              </Link>
            );
          })}
          <Link href="/customer/vehicles/new" className="rounded-xl border border-dashed border-input p-4 font-medium hover:border-ring">
            Добавить ТС
          </Link>
        </div>
      </section>
    </div>
  );
}

export function VehicleScreen({ id, booking = false }: { id: string; booking?: boolean }) {
  const { gateway } = useSession();
  const vehicle = useQuery(vehicleQuery(gateway, id));

  return (
    <div className="mx-auto max-w-2xl space-y-4">
      <Link href={booking ? vehiclePath(id) : "/customer"} className={link}>Назад</Link>
      <ReadState pending={vehicle.isPending} error={vehicle.error} retry={() => void vehicle.refetch()}>
        {vehicle.data && (booking
          ? <BookingForm vehicleId={id} vehicleLabel={`${vehicle.data.brand} · ${vehicle.data.number}`} />
          : <VehicleCard vehicle={vehicle.data} />)}
      </ReadState>
    </div>
  );
}

function VehicleCard({ vehicle }: { vehicle: Vehicle }) {
  const history = useHistory(vehicle.vehicle_id);
  return (
    <>
      <section className={`${panel} space-y-3`}>
        <h1>{vehicle.brand}</h1>
        <p>{vehicle.number}</p>
        <p className="text-sm text-muted-foreground">VIN {vehicle.vin}</p>
        <Button asChild><Link href={`${vehiclePath(vehicle.vehicle_id)}/booking`}>Записаться на сервис</Link></Button>
      </section>
      <VehiclePhotos vehicle={vehicle} />
      <section className={`${panel} space-y-3`}>
        <h2 className="text-lg font-semibold">История работ</h2>
        <ReadState pending={history.isPending} error={history.error} retry={() => void history.refetch()}>
          {history.data?.length ? <OrderList orders={history.data} /> : <p>Работ по этому ТС пока нет.</p>}
        </ReadState>
      </section>
    </>
  );
}

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
        <Status status={order.status} />
      </div>
      <Link className={link} href={vehiclePath(order.vehicle_id)}>
        {vehicle ? `${vehicle.brand} · ${vehicle.number}` : "Карточка ТС"}
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
