"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useSession } from "@/common/session/view";
import { CustomerForm } from "../../shared/forms/view";
import { vehiclePath } from "../../shared/routes";
import { createVehicle, vehicleMakesQuery } from "./controller";
import { describeVehicleFormError, readVehicleForm, validateVehicle, vehicleFields } from "./form";

export function VehicleForm() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { gateway } = useSession();
  const catalog = useQuery(vehicleMakesQuery(gateway));
  const makes = catalog.data ?? [];

  function submitVehicle(form: FormData) {
    const body = readVehicleForm(form, makes);
    if (!body) return Promise.reject(new Error("Vehicle make is missing"));

    return createVehicle(gateway, queryClient, body).then((id) => router.replace(vehiclePath(id)));
  }

  return (
    <CustomerForm title="Добавить ТС" action="Добавить ТС"
      fields={vehicleFields({
        makes,
        catalogPending: catalog.isPending,
        catalogLoaded: catalog.isSuccess,
        catalogError: catalog.isError,
        retryCatalog: () => void catalog.refetch(),
      })}
      onSubmit={submitVehicle} submitDisabled={!makes.length}
      validate={(form) => validateVehicle(form, makes)}
      describeError={describeVehicleFormError} />
  );
}
