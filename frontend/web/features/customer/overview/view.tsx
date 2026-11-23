"use client";

import Link from "next/link";
import { useSession } from "@/common/session/view";
import { Button } from "@/components/ui/button";
import { ReadState, panel } from "../shared/view";
import { newVehiclePath, vehicleBookingPath, vehiclePath } from "../shared/routes";
import { OrderList } from "../shared/orders/view";
import { useCustomerOverview } from "./controller";

export function CustomerOverview({ choosing = false }: { choosing?: boolean }) {
  const { gateway } = useSession();
  const { history, vehicleList } = useCustomerOverview(gateway, !choosing);
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
              <Link key={vehicle.vehicle_id} href={choosing ? vehicleBookingPath(vehicle.vehicle_id) : path} className={`${panel} hover:border-ring`}>
                <span className="block font-medium">{vehicle.brand}</span>
                <span className="text-sm text-muted-foreground">{vehicle.number}</span>
              </Link>
            );
          })}
          <Link href={newVehiclePath} className="rounded-xl border border-dashed border-input p-4 font-medium hover:border-ring">
            Добавить ТС
          </Link>
        </div>
      </section>
    </div>
  );
}
