"use client";

import { useRef, useState, type FormEvent } from "react";
import { GatewayError } from "@/lib/api/client";
import { isVehiclePlate, isVehicleVin } from "@/lib/forms/masks";

export type FieldErrors = Record<string, string | undefined>;
export type SubmitCustomerForm = (form: FormData) => Promise<void>;

function fieldValidation(form: FormData) {
  const errors: FieldErrors = {};

  for (const field of ["number", "brand", "vin", "service", "date"]) {
    if (form.has(field) && !fieldValue(form, field)) errors[field] = "Заполните это поле.";
  }

  const number = fieldValue(form, "number");
  const vin = fieldValue(form, "vin");

  if (number && !isVehiclePlate(number)) errors.number = "Введите госномер целиком, например А123ВС77.";
  if (vin && !isVehicleVin(vin)) errors.vin = "VIN состоит из 17 латинских букв и цифр. I, O и Q не используются.";

  return errors;
}

export function fieldValue(form: FormData, field: string) {
  return String(form.get(field) ?? "").trim();
}

export function useCustomerForm(send: SubmitCustomerForm, validate?: (form: FormData) => FieldErrors) {
  const submitting = useRef(false);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");
  const [errors, setErrors] = useState<FieldErrors>({});

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting.current) return;

    const data = new FormData(event.currentTarget);
    const invalid = { ...fieldValidation(data), ...validate?.(data) };
    setMessage("");
    setErrors(invalid);

    if (Object.keys(invalid).length) return;

    submitting.current = true;
    setPending(true);

    void send(data).catch((error: unknown) => {
      if (error instanceof GatewayError && error.code === "plate_taken") {
        setErrors({ number: "Этот номер уже зарегистрирован в СТО — обратитесь к мастеру." });
        return;
      }

      if (error instanceof GatewayError && error.code === "validation_failed") {
        setErrors(Object.fromEntries(Object.entries(error.fields ?? {}).map(([field, reason]) => [
          field, reason === "required" ? "Заполните это поле." : "Проверьте значение поля.",
        ])));
        setMessage("Проверьте данные формы.");
        return;
      }

      setMessage("Не удалось сохранить данные. Попробуйте ещё раз.");
    }).finally(() => {
      submitting.current = false;
      setPending(false);
    });
  }

  return { pending, message, errors, submit };
}
