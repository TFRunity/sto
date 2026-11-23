import { IMaskInput } from "react-imask";
import type { FieldErrors } from "@/common/hooks/use-form";
import { readFormText } from "@/common/utils/form-data";
import { Button } from "@/components/ui/button";
import { GatewayError } from "@/lib/api/client";
import { isVehiclePlate, isVehicleVin, vehiclePlateMask, vehicleVinMask } from "@/lib/forms/masks";
import { type FormField, SuggestionField } from "../../shared/forms/view";
import type { VehicleMake } from "./controller";
import { findMake, matchMakes } from "./search/controller";

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

export function vehicleFields({ makes, catalogPending, catalogLoaded, catalogError, retryCatalog }: {
  makes: VehicleMake[];
  catalogPending: boolean;
  catalogLoaded: boolean;
  catalogError: boolean;
  retryCatalog: () => void;
}): FormField[] {
  return [
    { name: "number", required: true, render: (error) => (
        <VehicleIdentifierField name="number" label="Госномер" placeholder="А123ВС77"
          hint="Кириллица и цифры, код региона из двух или трёх цифр." error={error} />
    ) },
    { name: "brand", required: true, render: (error) => <>
        <SuggestionField name="brand" label="Марка" placeholder="Начните вводить марку"
          findOptions={(input) => matchMakes(makes, input).map(({ name }) => name)}
          noMatches="Подходящих марок нет." disabled={!makes.length} error={error} />
        {catalogPending && <p role="status" className="text-sm">Загружаем марки…</p>}
        {catalogLoaded && !makes.length && <p role="alert">Справочник марок пока пуст.</p>}
        {catalogError && (
          <div className="space-y-2">
            <p role="alert">Не удалось загрузить справочник марок.</p>
            <Button type="button" variant="outline" onClick={retryCatalog}>Повторить</Button>
          </div>
        )}
    </> },
    { name: "vin", required: true, render: (error) => (
        <VehicleIdentifierField name="vin" label="VIN" placeholder="JT2BF22K1X0123456"
          hint="17 символов, латинские буквы и цифры без I, O и Q." error={error} />
    ) },
  ];
}

export function validateVehicle(form: FormData, makes: VehicleMake[]): FieldErrors {
  const errors: FieldErrors = {};
  const number = readFormText(form, "number");
  const vin = readFormText(form, "vin");

  if (!findMake(makes, readFormText(form, "brand"))) errors.brand = "Выберите марку из справочника.";
  if (number && !isVehiclePlate(number)) errors.number = "Введите госномер целиком, например А123ВС77.";
  if (vin && !isVehicleVin(vin)) errors.vin = "VIN состоит из 17 латинских букв и цифр. I, O и Q не используются.";
  return errors;
}

export function readVehicleForm(form: FormData, makes: VehicleMake[]) {
  const brand = findMake(makes, readFormText(form, "brand"));
  if (!brand) return undefined;

  return {
    number: readFormText(form, "number"),
    brand: brand.name,
    vin: readFormText(form, "vin"),
  };
}

export function describeVehicleFormError(error: unknown) {
  if (error instanceof GatewayError && error.code === "plate_taken") {
    return { fields: { number: "Этот номер уже зарегистрирован в СТО — обратитесь к мастеру." } };
  }
}
