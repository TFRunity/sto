"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { IMaskInput } from "react-imask";
import { Button } from "@/components/ui/button";
import { readFormText } from "@/common/utils/form-data";
import { useSession } from "@/features/auth/session-provider";
import { isVehiclePlate, isVehicleVin, vehiclePlateMask, vehicleVinMask } from "@/lib/forms/masks";
import { createVehicle, customerVehicleListKey, vehicleMakesQuery } from "./vehicle-api";
import { vehiclePath } from "./customer-routes";
import { CustomerForm, type FormField } from "./customer-form";
import { SuggestionField } from "./suggestion-field";
import { findMake, matchMakes } from "./suggestions";
import type { FieldErrors } from "@/common/hooks/use-form";
import { GatewayError } from "@/lib/api/client";

function VehicleIdentifierField({ name, label, error, placeholder, hint }: {
  name: "number" | "vin";
  label: string;
  error?: string;
  placeholder: string;
  hint: string;
}) {
  const mask = name === "number" ? vehiclePlateMask : vehicleVinMask;

  return (
    <div className="space-y-1">
      <label className="text-sm font-medium" htmlFor={name}>{label}</label>
      <IMaskInput
        {...mask}
        id={name}
        name={name}
        required
        autoCapitalize="characters"
        spellCheck={false}
        placeholder={placeholder}
        aria-invalid={Boolean(error)}
        aria-describedby={`${name}-feedback`}
        className="h-10 w-full rounded-md border border-input px-3 focus-visible:outline-2 focus-visible:outline-ring aria-invalid:border-destructive"
      />
      <p id={`${name}-feedback`} role={error ? "alert" : undefined}
        className={error ? "text-sm text-destructive" : "text-sm text-muted-foreground"}>
        {error ?? hint}
      </p>
    </div>
  );
}

export function VehicleForm() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { gateway } = useSession();
  const catalog = useQuery(vehicleMakesQuery(gateway));
  const makes = catalog.data ?? [];
  const findBrand = (form: FormData) => findMake(makes, readFormText(form, "brand"));

  function validateVehicle(form: FormData): FieldErrors {
    const errors: FieldErrors = {};
    const number = readFormText(form, "number");
    const vin = readFormText(form, "vin");

    if (!findBrand(form)) errors.brand = "Выберите марку из справочника.";
    if (number && !isVehiclePlate(number)) errors.number = "Введите госномер целиком, например А123ВС77.";
    if (vin && !isVehicleVin(vin)) errors.vin = "VIN состоит из 17 латинских букв и цифр. I, O и Q не используются.";
    return errors;
  }

  function submitVehicle(form: FormData) {
    const brand = findBrand(form);
    if (!brand) return Promise.reject(new Error("Vehicle make is missing"));

    return createVehicle(gateway, {
      number: readFormText(form, "number"),
      brand: brand.name,
      vin: readFormText(form, "vin"),
    }).then((id) => {
      void queryClient.invalidateQueries({ queryKey: customerVehicleListKey });
      router.replace(vehiclePath(id));
    });
  }

  const fields: FormField[] = [
    { name: "number", required: true, render: (error) => (
        <VehicleIdentifierField name="number" label="Госномер" placeholder="А123ВС77"
          hint="Кириллица и цифры, код региона из двух или трёх цифр." error={error} />
    ) },
    { name: "brand", required: true, render: (error) => <>
        <SuggestionField name="brand" label="Марка" placeholder="Начните вводить марку"
          findOptions={(input) => matchMakes(makes, input).map(({ name }) => name)}
          noMatches="Подходящих марок нет." disabled={!makes.length} error={error} />
        {catalog.isPending && <p role="status" className="text-sm">Загружаем марки…</p>}
        {catalog.isSuccess && !makes.length && <p role="alert">Справочник марок пока пуст.</p>}
        {catalog.isError && (
          <div className="space-y-2">
            <p role="alert">Не удалось загрузить справочник марок.</p>
            <Button type="button" variant="outline" onClick={() => void catalog.refetch()}>Повторить</Button>
          </div>
        )}
    </> },
    { name: "vin", required: true, render: (error) => (
        <VehicleIdentifierField name="vin" label="VIN" placeholder="JT2BF22K1X0123456"
          hint="17 символов, латинские буквы и цифры без I, O и Q." error={error} />
    ) },
  ];

  return (
    <CustomerForm title="Добавить ТС" action="Добавить ТС"
      fields={fields} onSubmit={submitVehicle} submitDisabled={!makes.length}
      validate={validateVehicle}
      describeError={(error) => {
        if (error instanceof GatewayError && error.code === "plate_taken") {
          return { fields: { number: "Этот номер уже зарегистрирован в СТО — обратитесь к мастеру." } };
        }
      }} />
  );
}
