import { GatewayError } from "@/lib/api/client";
import { normalizeRussianPhone } from "@/lib/forms/masks";
import { WebRoleError } from "./session";

type AuthField = "name" | "phone" | "password";
export type AuthErrors = Partial<Record<AuthField, string>>;

const fields: AuthField[] = ["name", "phone", "password"];
const fieldMessages: Record<string, string> = {
  required: "Заполните это поле.",
  too_long: "Пароль должен занимать не более 72 байт.",
  format: "Проверьте формат значения.",
  minLength: "Значение слишком короткое.",
  maxLength: "Значение слишком длинное.",
};

export function readAuthForm(form: FormData, register: boolean) {
  const name = String(form.get("name") ?? "").trim();
  const phone = normalizeRussianPhone(String(form.get("phone") ?? ""));
  const password = String(form.get("password") ?? "");
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
        const invalid: AuthErrors = {};
        for (const field of fields) {
          const reason = error.fields?.[field];
          if (reason) invalid[field] = fieldMessages[reason] ?? "Проверьте значение поля.";
        }
        return Object.keys(invalid).length ? { fields: invalid } : { message: "Проверьте данные формы." };
      }
    }
  }

  return { message: "Не удалось связаться с сервисом. Попробуйте ещё раз." };
}
