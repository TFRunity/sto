export function readFormText(form: FormData, field: string, { trim = true } = {}) {
  const value = String(form.get(field) ?? "");
  return trim ? value.trim() : value;
}
