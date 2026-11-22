"use client";

import { useRouter } from "next/navigation";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { IMaskInput } from "react-imask";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useSession } from "@/features/auth/session-provider";
import { vehiclePlateMask, vehicleVinMask } from "@/lib/forms/masks";
import {
  createBooking, createVehicle, customerOrderHistoryKey, customerVehicleListKey, vehicleMakesQuery,
} from "./customer-api";
import { fieldValue, useCustomerForm, type FieldErrors, type SubmitCustomerForm } from "./use-customer-form";
import { VehicleMakeField } from "./vehicle-make-field";
import { findMake } from "./vehicle-makes";

function CustomerForm({ title, action, onSubmit, validate, submitDisabled, children }: {
  title: string;
  action: string;
  onSubmit: SubmitCustomerForm;
  validate?: (form: FormData) => FieldErrors;
  submitDisabled?: boolean;
  children: (errors: FieldErrors) => ReactNode;
}) {
  const { pending, message, errors, submit } = useCustomerForm(onSubmit, validate);

  return (
    <form onSubmit={submit} aria-busy={pending} className="mx-auto max-w-lg space-y-4 rounded-xl border bg-card p-5 sm:p-6">
      <h1>{title}</h1>
      <fieldset disabled={pending} className="space-y-4">
        {children(errors)}
        {message && <p role="alert" className="text-sm text-destructive">{message}</p>}
        <Button type="submit" disabled={submitDisabled}>{pending ? "Отправляем…" : action}</Button>
      </fieldset>
    </form>
  );
}

function Field({ name, label, error, type = "text", placeholder }: {
  name: string; label: string; error?: string; type?: string; placeholder?: string;
}) {
  return (
    <div className="space-y-1">
      <label className="text-sm font-medium" htmlFor={name}>{label}</label>
      <Input id={name} name={name} type={type} placeholder={placeholder} required
        aria-invalid={Boolean(error)} aria-describedby={error ? `${name}-error` : undefined} />
      {error && <p role="alert" id={`${name}-error`} className="text-sm text-destructive">{error}</p>}
    </div>
  );
}

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
  const findBrand = (form: FormData) => findMake(makes, fieldValue(form, "brand"));

  function submitVehicle(form: FormData) {
    const brand = findBrand(form);
    if (!brand) return Promise.reject(new Error("Vehicle make is missing"));

    return createVehicle(gateway, {
      number: fieldValue(form, "number"),
      brand: brand.name,
      vin: fieldValue(form, "vin"),
    }).then((id) => {
      void queryClient.invalidateQueries({ queryKey: customerVehicleListKey });
      router.replace(`/customer/vehicles/${id}`);
    });
  }

  return (
    <CustomerForm title="Добавить ТС" action="Добавить ТС"
      onSubmit={submitVehicle} submitDisabled={!makes.length}
      validate={(form) => findBrand(form) ? {} : { brand: "Выберите марку из справочника." }}>
      {(errors) => <>
        <VehicleIdentifierField name="number" label="Госномер" placeholder="А123ВС77"
          hint="Кириллица и цифры, код региона из двух или трёх цифр." error={errors.number} />
        <VehicleMakeField makes={makes} disabled={!makes.length} error={errors.brand} />
        {catalog.isPending && <p role="status" className="text-sm">Загружаем марки…</p>}
        {catalog.isSuccess && !makes.length && <p role="alert">Справочник марок пока пуст.</p>}
        {catalog.isError && (
          <div className="space-y-2">
            <p role="alert">Не удалось загрузить справочник марок.</p>
            <Button type="button" variant="outline" onClick={() => void catalog.refetch()}>Повторить</Button>
          </div>
        )}
        <VehicleIdentifierField name="vin" label="VIN" placeholder="JT2BF22K1X0123456"
          hint="17 символов, латинские буквы и цифры без I, O и Q." error={errors.vin} />
      </>}
    </CustomerForm>
  );
}

export function BookingForm({ vehicleId, vehicleLabel }: { vehicleId: string; vehicleLabel: string }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { gateway } = useSession();
  return (
    <CustomerForm title="Записаться на сервис" action="Отправить заявку" onSubmit={(form) => {
      const comment = fieldValue(form, "comment");

      return createBooking(gateway, {
        vehicle_id: vehicleId,
        service: fieldValue(form, "service"),
        date: fieldValue(form, "date"),
        ...(comment ? { comment } : {}),
      }).then((id) => {
        void queryClient.invalidateQueries({ queryKey: customerOrderHistoryKey });
        router.replace(`/customer/orders/${id}`);
      });
    }}>
      {(errors) => <>
        <div><p className="text-sm font-medium">Автомобиль</p><p>{vehicleLabel}</p></div>
        <Field name="service" label="Желаемая услуга" placeholder="Замена масла" error={errors.service} />
        <Field name="date" label="Желаемая дата" type="date" error={errors.date} />
        <div className="space-y-1">
          <label className="text-sm font-medium" htmlFor="comment">
            Комментарий <span className="font-normal text-muted-foreground">(необязательно)</span>
          </label>
          <textarea className="block w-full rounded-md border border-input p-2 focus-visible:outline-2 focus-visible:outline-ring"
            id="comment" name="comment" rows={2} aria-invalid={Boolean(errors.comment)}
            aria-describedby={errors.comment ? "comment-error" : undefined} />
          {errors.comment && <p role="alert" id="comment-error" className="text-sm text-destructive">{errors.comment}</p>}
        </div>
      </>}
    </CustomerForm>
  );
}
