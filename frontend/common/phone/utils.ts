export function normalizeRussianPhone(value: string) {
  if (!/^[+\d()\s-]+$/.test(value)) return undefined;

  const digits = value.replace(/\D/g, "");
  return /^[78]\d{10}$/.test(digits) ? `+7${digits.slice(1)}` : undefined;
}
