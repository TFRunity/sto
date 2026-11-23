const rubles = new Intl.NumberFormat("ru-RU", { style: "currency", currency: "RUB" });

export function formatRubles(amount: string) {
  return rubles.format(Number(amount));
}
