import { formatCalendarDate } from "@/common/utils/date";
import { readFormText } from "@/common/utils/form-data";
import { Input } from "@/components/ui/input";
import { type FormItem, SuggestionField } from "../shared/forms/view";
import { matchServiceNames } from "./search/controller";

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

export function bookingFields({ vehicleLabel, serviceNames, catalogError, catalogLoaded, now }: {
  vehicleLabel: string;
  serviceNames: string[];
  catalogError: boolean;
  catalogLoaded: boolean;
  now: Date;
}): FormItem[] {
  return [
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
            {catalogError && (
              <p role="status" className="text-sm text-muted-foreground">Каталог недоступен. Введите услугу вручную.</p>
            )}
            {catalogLoaded && !serviceNames.length && (
              <p role="status" className="text-sm text-muted-foreground">Каталог пока пуст. Введите услугу вручную.</p>
            )}
        </> },
        { name: "date", required: true, render: (error) => (
            <Field name="date" label="Желаемая дата" type="date"
              min={formatCalendarDate(now, "Europe/Moscow")} error={error} />
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
}

export const bookingFieldMessages = { past_date: "Выберите сегодня или более позднюю дату." };

export function readBookingForm(form: FormData, vehicleId: string) {
  const comment = readFormText(form, "comment");

  return {
    vehicle_id: vehicleId,
    service: readFormText(form, "service"),
    date: readFormText(form, "date"),
    ...(comment ? { comment } : {}),
  };
}
