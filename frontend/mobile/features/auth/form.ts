import { normalizeRussianPhone } from "../../../common/phone/utils";

export function readLoginForm(phone: string, password: string) {
  const normalizedPhone = normalizeRussianPhone(phone);
  if (!normalizedPhone) return { error: "Введите полный номер телефона." };
  if (!password) return { error: "Введите пароль." };

  return { body: { phone: normalizedPhone, password } };
}
