"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { Input } from "@/components/ui/input";
import { readFormText } from "@/common/utils/form-data";
import { useSession } from "@/features/auth/session-provider";
import { formatCalendarDate } from "@/common/utils/date";
import { createBooking, customerOrderHistoryKey } from "./orders-api";
import { bookingServiceNamesQuery } from "./services-api";
import { orderPath } from "./customer-routes";
import { CustomerForm, type FormItem } from "./customer-form";
import { SuggestionField } from "./suggestion-field";
import { matchServiceNames } from "./suggestions";

function Field({ name, label, error, type = "text", placeholder, min }: {
  name: string; label: string; error?: string; type?: string; placeholder?: string; min?: string;
}) {
  return (
    <div className="space-y-1">
      <label className="text-sm font-medium" htmlFor={name}>{label}</label>
      <Input id={name} name={name} type={type} placeholder={placeholder} min={min} required
        aria-invalid={Boolean(error)} aria-describedby={error ? `${name}-error` : undefined} />
      {error && <p role="alert" id={`${name}-error`} className="text-sm text-destructive">{error}</p>}
    </div>
  );
}

export function BookingForm({ vehicleId, vehicleLabel }: { vehicleId: string; vehicleLabel: string }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { gateway } = useSession();
  const services = useQuery(bookingServiceNamesQuery(gateway));
  const serviceNames = services.data ?? [];

  const fields: FormItem[] = [
    { name: "vehicle", render: () => (
        <div><p className="text-sm font-medium">Автомобиль</p><p>{vehicleLabel}</p></div>
    ) },
    {
      name: "visit-details",
      row: [
        { name: "service", required: true, render: (error) => <>
            <SuggestionField name="service" label="Желаемая услуга" placeholder="Замена масла"
              findOptions={(input) => matchServiceNames(serviceNames, input)}
              noMatches={serviceNames.length ? "Совпадений нет. Можно ввести свою услугу." : undefined}
              error={error} />
            {services.isError && (
              <p role="status" className="text-sm text-muted-foreground">Каталог недоступен. Введите услугу вручную.</p>
            )}
            {services.isSuccess && !serviceNames.length && (
              <p role="status" className="text-sm text-muted-foreground">Каталог пока пуст. Введите услугу вручную.</p>
            )}
        </> },
        { name: "date", required: true, render: (error) => (
            <Field name="date" label="Желаемая дата" type="date"
              min={formatCalendarDate(new Date(), "Europe/Moscow")} error={error} />
        ) },
      ],
    },
    { name: "comment", render: (error) => (
        <div className="space-y-1">
          <label className="text-sm font-medium" htmlFor="comment">
            Комментарий <span className="font-normal text-muted-foreground">(необязательно)</span>
          </label>
          <textarea className="block w-full rounded-md border border-input p-2 focus-visible:outline-2 focus-visible:outline-ring"
            id="comment" name="comment" rows={2} aria-invalid={Boolean(error)}
            aria-describedby={error ? "comment-error" : undefined} />
          {error && <p role="alert" id="comment-error" className="text-sm text-destructive">{error}</p>}
        </div>
    ) },
  ];

  function submitBooking(form: FormData) {
    const comment = readFormText(form, "comment");

    return createBooking(gateway, {
      vehicle_id: vehicleId,
      service: readFormText(form, "service"),
      date: readFormText(form, "date"),
      ...(comment ? { comment } : {}),
    }).then((id) => {
      void queryClient.invalidateQueries({ queryKey: customerOrderHistoryKey });
      router.replace(orderPath(id));
    });
  }

  return (
    <CustomerForm title="Записаться на сервис" action="Отправить заявку"
      fields={fields} onSubmit={submitBooking}
      fieldMessages={{ past_date: "Выберите сегодня или более позднюю дату." }} />
  );
}
