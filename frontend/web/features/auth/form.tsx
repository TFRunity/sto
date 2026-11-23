import { Eye, EyeOff } from "lucide-react";
import { IMaskInput } from "react-imask";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { preparePhoneInput, russianPhoneMask, russianPhonePlaceholder } from "@/lib/forms/masks";
import { readFormText } from "@/common/utils/form-data";
import { validationMessages } from "@/common/utils/validation-messages";
import { GatewayError } from "@/lib/api/client";
import { normalizeRussianPhone } from "@shared/phone/utils";
import { WebRoleError } from "@/common/session/session";

export type AuthField = "name" | "phone" | "password";
export type AuthErrors = Partial<Record<AuthField, string>>;

const fields: AuthField[] = ["name", "phone", "password"];
const fieldMessages: Record<string, string> = {
  too_long: "Пароль должен занимать не более 72 байт.",
  format: "Проверьте формат значения.",
  minLength: "Значение слишком короткое.",
  maxLength: "Значение слишком длинное.",
};

export function readAuthForm(form: FormData, register: boolean) {
  const name = readFormText(form, "name");
  const phone = normalizeRussianPhone(readFormText(form, "phone", { trim: false }));
  const password = readFormText(form, "password", { trim: false });
  const errors: AuthErrors = {};

  if (register && !name) errors.name = "Укажите имя.";
  if (!phone) errors.phone = "Введите телефон полностью, в формате +7.";
  if (!password) {
    errors.password = "Введите пароль.";
  } else if (register && password.length < 8) {
    errors.password = "Используйте не менее 8 символов.";
  } else if (register && new TextEncoder().encode(password).length > 72) {
    // bcrypt считает байты UTF-8. Кириллица и эмодзи могут занимать больше одного байта.
    errors.password = fieldMessages.too_long;
  }

  return !phone || Object.keys(errors).length ? { errors } : { body: { name, phone, password } };
}

export function describeAuthError(error: unknown): { fields?: AuthErrors; message?: string } {
  if (error instanceof WebRoleError) return { message: "Веб-кабинет доступен клиенту и директору." };

  if (error instanceof GatewayError) {
    switch (error.code) {
      case "invalid_credentials": return { message: "Проверьте телефон и пароль." };
      case "phone_taken": return { fields: { phone: "Этот телефон уже зарегистрирован." } };
      case "validation_failed": {
        const invalid = validationMessages(error.fields, fieldMessages, fields);
        return Object.keys(invalid).length ? { fields: invalid } : { message: "Проверьте данные формы." };
      }
    }
  }

  return { message: "Не удалось связаться с сервисом. Попробуйте ещё раз." };
}

const inputClass = "h-12 w-full rounded-lg border border-input bg-white px-3.5 text-base shadow-xs";
const maskFocusClass = "focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none aria-invalid:border-destructive";

export function authFields({ register, errors, clearField, showPassword, togglePassword }: {
  register: boolean;
  errors: AuthErrors;
  clearField: (field: AuthField) => void;
  showPassword: boolean;
  togglePassword: () => void;
}) {
  return [
    { name: "name", render: () => register && (
          <div className="space-y-1.5">
            <label className="font-medium" htmlFor="name">Имя</label>
            <Input id="name" name="name" required autoComplete="name" className={inputClass}
              placeholder="Иван Иванов" aria-invalid={Boolean(errors.name)}
              aria-describedby={errors.name ? "name-error" : undefined}
              onChange={() => clearField("name")} />
            <FieldFeedback name="name" error={errors.name} />
          </div>
    ) },
    { name: "phone", render: () => (
        <div className="space-y-1.5">
          <label className="font-medium" htmlFor="phone">Телефон</label>
          <IMaskInput id="phone" name="phone" type="text" inputMode="tel" autoComplete="tel"
            required mask={russianPhoneMask} prepare={preparePhoneInput}
            placeholder={russianPhonePlaceholder} className={`${inputClass} ${maskFocusClass}`}
            aria-invalid={Boolean(errors.phone)} aria-describedby={errors.phone ? "phone-error" : undefined}
            onAccept={() => clearField("phone")} />
          <FieldFeedback name="phone" error={errors.phone} />
        </div>
    ) },
    { name: "password", render: () => (
        <div className="space-y-1.5">
          <label className="font-medium" htmlFor="password">Пароль</label>
          <div className="relative">
            <Input id="password" name="password" type={showPassword ? "text" : "password"}
              className={`${inputClass} pr-12`} required
              autoComplete={register ? "new-password" : "current-password"}
              aria-invalid={Boolean(errors.password)}
              aria-describedby={errors.password ? "password-error" : register ? "password-hint" : undefined}
              onChange={() => clearField("password")} />
            <Button type="button" variant="ghost" size="icon" className="absolute top-1.5 right-1.5"
              aria-label={showPassword ? "Скрыть пароль" : "Показать пароль"}
              onClick={togglePassword}>
              {showPassword ? <EyeOff aria-hidden="true" /> : <Eye aria-hidden="true" />}
            </Button>
          </div>
          <FieldFeedback name="password" error={errors.password} />
          {register && !errors.password && (
            <p id="password-hint" className="text-sm text-muted-foreground">Не менее 8 символов.</p>
          )}
        </div>
    ) },
  ];
}

function FieldFeedback({ name, error }: { name: string; error?: string }) {
  return error
    ? <p role="alert" id={`${name}-error`} className="text-sm text-destructive">{error}</p>
    : null;
}
