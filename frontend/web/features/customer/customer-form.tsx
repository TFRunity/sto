import { Fragment, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
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
