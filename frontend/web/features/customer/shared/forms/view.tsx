"use client";

import { Fragment, useId, useState, type KeyboardEvent, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useForm, type FieldErrors, type FormFeedback, type SubmitForm } from "@/common/hooks/use-form";
import { readFormText } from "@/common/utils/form-data";
import { validationMessage, validationMessages } from "@/common/utils/validation-messages";
import { GatewayError } from "@/lib/api/client";

export type FormField = {
  name: string;
  required?: boolean;
  render: (error?: string) => ReactNode;
};

export type FormItem = FormField | {
  name: string;
  row: readonly FormField[];
};

export function CustomerForm({ title, action, onSubmit, validate, submitDisabled, fields, describeError, fieldMessages }: {
  title: string;
  action: string;
  onSubmit: SubmitForm;
  validate?: (form: FormData) => FieldErrors;
  submitDisabled?: boolean;
  fields: readonly FormItem[];
  describeError?: (error: unknown) => FormFeedback | undefined;
  fieldMessages?: Readonly<Record<string, string>>;
}) {
  const { pending, message, errors, submit } = useForm({
    send: onSubmit,
    validate(form) {
      const missing: FieldErrors = {};
      for (const { name, required } of fields.flatMap((item) => "row" in item ? item.row : [item])) {
        if (required && !readFormText(form, name)) missing[name] = validationMessage("required");
      }
      return { ...missing, ...validate?.(form) };
    },
    describeError(error) {
      const feedback = describeError?.(error);
      if (feedback) return feedback;

      if (error instanceof GatewayError && error.code === "validation_failed") {
        return {
          fields: validationMessages(error.fields, fieldMessages),
          message: "Проверьте данные формы.",
        };
      }
      return { message: "Не удалось сохранить данные. Попробуйте ещё раз." };
    },
  });

  return (
    <form onSubmit={submit} aria-busy={pending} className="mx-auto max-w-lg space-y-4 rounded-xl border bg-card p-5 sm:p-6">
      <h1>{title}</h1>
      <fieldset disabled={pending} className="space-y-4">
        {fields.map((item) => "row" in item ? (
          <div key={item.name} className="grid items-start gap-4 sm:auto-cols-fr sm:grid-flow-col">
            {item.row.map(({ name, render }) => (
              <div key={name} className="min-w-0 space-y-1">{render(errors[name])}</div>
            ))}
          </div>
        ) : (
          <Fragment key={item.name}>{item.render(errors[item.name])}</Fragment>
        ))}
        {message && <p role="alert" className="text-sm text-destructive">{message}</p>}
        <Button type="submit" disabled={submitDisabled}>{pending ? "Отправляем…" : action}</Button>
      </fieldset>
    </form>
  );
}

export function SuggestionField({ name, label, placeholder, findOptions, noMatches, error, disabled }: {
  name: string;
  label: string;
  placeholder: string;
  findOptions: (input: string) => string[];
  noMatches?: string;
  error?: string;
  disabled?: boolean;
}) {
  const id = useId();
  const [value, setValue] = useState("");
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const options = findOptions(value);
  const expanded = open && !disabled && options.length > 0;

  function select(option: string) {
    setValue(option);
    setOpen(false);
    setActive(-1);
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Escape" && open) {
      event.preventDefault();
      setOpen(false);
      setActive(-1);
    } else if (event.key === "Enter" && expanded && options[active]) {
      event.preventDefault();
      select(options[active]);
    } else if (options.length && (event.key === "ArrowDown" || event.key === "ArrowUp")) {
      event.preventDefault();
      setOpen(true);
      setActive(event.key === "ArrowDown"
        ? Math.min(active + 1, options.length - 1)
        : active <= 0 ? options.length - 1 : active - 1);
    }
  }

  return (
    <div className="space-y-1">
      <label className="text-sm font-medium" htmlFor={id}>{label}</label>
      <Input
        id={id}
        name={name}
        role="combobox"
        autoComplete="off"
        aria-autocomplete="list"
        aria-expanded={expanded}
        aria-controls={expanded ? `${id}-options` : undefined}
        aria-activedescendant={expanded && options[active] ? `${id}-option-${active}` : undefined}
        aria-invalid={Boolean(error)}
        aria-describedby={error ? `${id}-error` : undefined}
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        required
        onFocus={() => setOpen(true)}
        onBlur={() => {
          setOpen(false);
          setActive(-1);
        }}
        onChange={(event) => {
          setValue(event.target.value);
          setActive(-1);
          setOpen(true);
        }}
        onKeyDown={handleKeyDown}
      />
      {expanded && (
        <ul id={`${id}-options`} role="listbox" aria-label={label}
          className="max-h-48 overflow-y-auto rounded-md border bg-card p-1 shadow-sm">
          {options.map((option, index) => (
            <li id={`${id}-option-${index}`} key={option} role="option"
              aria-selected={active === index}
              className="cursor-pointer rounded px-3 py-2 hover:bg-accent aria-selected:bg-accent"
              onMouseDown={(event) => {
                // Фокус остаётся в поле, пока пользователь выбирает вариант мышью.
                event.preventDefault();
              }}
              onClick={() => select(option)}>
              {option}
            </li>
          ))}
        </ul>
      )}
      {open && !disabled && noMatches && options.length === 0 && (
        <p role="status" className="text-sm text-muted-foreground">{noMatches}</p>
      )}
      {error && <p role="alert" id={`${id}-error`} className="text-sm text-destructive">{error}</p>}
    </div>
  );
}
