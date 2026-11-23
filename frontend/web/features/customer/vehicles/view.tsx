"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useSession } from "@/common/session/view";
import { Button } from "@/components/ui/button";
import type { components } from "@/lib/api/schema";
import { ReadState, link, panel } from "../shared/view";

import { vehicleBookingPath } from "../shared/routes";
import { useHistory } from "../shared/orders/controller";
import { OrderList } from "../shared/orders/view";
import { vehicleQuery } from "../shared/vehicles/controller";
import { VehicleCover } from "./cover/view";
import { VehiclePhotos } from "./photos/view";

type Vehicle = components["schemas"]["VehicleCard"];

export function VehicleScreen({ id }: { id: string }) {
  const { gateway } = useSession();
  const vehicle = useQuery(vehicleQuery(gateway, id));

  return (
    <div className="mx-auto max-w-2xl space-y-4">
      <Link href="/customer" className={link}>Назад</Link>
      <ReadState pending={vehicle.isPending} error={vehicle.error} retry={() => void vehicle.refetch()}>
        {vehicle.data && <VehicleCard vehicle={vehicle.data} />}
      </ReadState>
    </div>
  );
}

function VehicleCard({ vehicle }: { vehicle: Vehicle }) {
  const { gateway } = useSession();
  const history = useHistory(gateway, vehicle.vehicle_id);
  return (
    <>
      <section className={`${panel} space-y-3`}>
        <h1>{vehicle.brand}</h1>
        <p>{vehicle.number}</p>
        <p className="text-sm text-muted-foreground">VIN {vehicle.vin}</p>
        <VehicleCover id={vehicle.vehicle_id} hasCover={Boolean(vehicle.cover_photo_url)} />
        <Button asChild><Link href={vehicleBookingPath(vehicle.vehicle_id)}>Записаться на сервис</Link></Button>
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
