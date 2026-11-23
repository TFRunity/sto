export function validationMessage(reason: string) {
  return reason === "required" ? "Заполните это поле." : "Проверьте значение поля.";
}

export function validationMessages(
  fields: Readonly<Record<string, string>> = {},
  overrides: Readonly<Record<string, string>> = {},
  allowedFields?: readonly string[],
) {
  return Object.fromEntries(
    Object.entries(fields)
      .filter(([field, reason]) => !allowedFields || (reason && allowedFields.includes(field)))
      .map(([field, reason]) => [
        field, Object.hasOwn(overrides, reason) ? overrides[reason] : validationMessage(reason),
      ]),
  );
}
