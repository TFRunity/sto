"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useSession } from "@/common/session/view";
import { useRouter } from "next/navigation";
import { ReadState, link } from "../shared/view";
import { orderPath, vehiclePath } from "../shared/routes";
import { vehicleQuery } from "../shared/vehicles/controller";
import { vehicleLabel } from "../shared/utils";
import { VehicleCoverPhoto } from "../shared/vehicles/view";
import { createBooking } from "./controller";
import { CustomerForm } from "../shared/forms/view";
import { bookingServiceNamesQuery } from "./search/controller";
import { bookingFieldMessages, bookingFields, readBookingForm } from "./form";

export function BookingScreen({ id }: { id: string }) {
  const { gateway } = useSession();
  const vehicle = useQuery(vehicleQuery(gateway, id));

  return (
    <div className="mx-auto max-w-2xl space-y-4">
      <Link href={vehiclePath(id)} className={link}>Назад</Link>
      <ReadState pending={vehicle.isPending} error={vehicle.error} retry={() => void vehicle.refetch()}>
        {vehicle.data && (
          <div className="mx-auto max-w-lg space-y-4">
            {vehicle.data.cover_photo_url && (
              <div><VehicleCoverPhoto id={id} /></div>
            )}
            <BookingForm vehicleId={id} vehicleLabel={vehicleLabel(vehicle.data)} />
          </div>
        )}
      </ReadState>
    </div>
  );
}

function BookingForm({ vehicleId, vehicleLabel }: { vehicleId: string; vehicleLabel: string }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { gateway } = useSession();
  const services = useQuery(bookingServiceNamesQuery(gateway));

  return (
    <CustomerForm title="Записаться на сервис" action="Отправить заявку"
      fields={bookingFields({
        vehicleLabel,
        serviceNames: services.data ?? [],
        catalogError: services.isError,
        catalogLoaded: services.isSuccess,
        now: new Date(),
      })}
      onSubmit={(form) => createBooking(gateway, queryClient, readBookingForm(form, vehicleId))
        .then((id) => router.replace(orderPath(id)))}
      fieldMessages={bookingFieldMessages} />
  );
}
