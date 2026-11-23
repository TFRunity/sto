import { readFormText } from "@/common/utils/form-data";
import { validationMessages } from "@/common/utils/validation-messages";
import { GatewayError } from "@/lib/api/client";
import { normalizeRussianPhone } from "../../../common/phone/utils";
import { WebRoleError } from "./session";

type AuthField = "name" | "phone" | "password";
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
