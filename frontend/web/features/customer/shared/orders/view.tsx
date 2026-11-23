import Link from "next/link";
import { orderPath, vehiclePath } from "../routes";
import { link, panel } from "../view";
import { vehicleLabel, type VehicleLabelData } from "../utils";
import type { HistoryItem } from "./controller";

const statusLabels = { requested: "Заявка", active: "В работе", paid: "Оплачен", cancelled: "Отменён" };
export const stageLabels = { принят: "Принят", в_работе: "В работе", готов: "Готов", выдан: "Выдан" };
const statusColors = {
  requested: "bg-accent text-accent-foreground",
  active: "bg-[#faeeda] text-[#854f0b]",
  paid: "bg-[#eaf3de] text-[#3b6d11]",
  cancelled: "bg-[#fcebeb] text-destructive",
};

export function OrderStatus({ status }: { status: HistoryItem["status"] }) {
  return <span className={`rounded-md px-2.5 py-1 text-sm ${statusColors[status]}`}>{statusLabels[status]}</span>;
}

const dateFormat = new Intl.DateTimeFormat("ru-RU", { dateStyle: "medium" });

export function OrderList({ orders, vehicles }: { orders: HistoryItem[]; vehicles?: ReadonlyMap<string, VehicleLabelData> }) {
  return (
    <ul className="space-y-3">
      {orders.map((order) => {
        const vehicle = vehicles?.get(order.vehicle_id);
        return (
          <li key={order.id} className={`${panel} flex flex-wrap items-start justify-between gap-3`}>
            <div className="space-y-1">
              <Link href={orderPath(order.id)} className={link}>
                {vehicle ? vehicleLabel(vehicle) : `Заказ ${order.id.slice(0, 8)}`}
              </Link>
              <p className="text-sm text-muted-foreground">
                {dateFormat.format(new Date(order.created_at))}
                {order.status === "requested" ? "" : ` · ${stageLabels[order.stage]}`}
              </p>
              <Link href={vehiclePath(order.vehicle_id)} className={`text-sm ${link}`}>Карточка ТС</Link>
            </div>
            <OrderStatus status={order.status} />
          </li>
        );
      })}
    </ul>
  );
}
