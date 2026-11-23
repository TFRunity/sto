import type { components } from "@shared/api/schema";
import type { FieldErrors } from "@/common/hooks/use-form";
import { readFormText } from "@/common/utils/form-data";

export type CatalogKind = "inventory" | "services";

type InventoryInput = components["schemas"]["InventoryInput"];
type ServiceInput = components["schemas"]["ServiceInput"];

export type CatalogInput =
  | { kind: "inventory"; body: InventoryInput }
  | { kind: "services"; body: ServiceInput };

export function catalogFields(kind: CatalogKind) {
  const fields = [
    { name: "name", label: "Название", inputMode: "text" },
    { name: "price", label: "Цена, ₽", inputMode: "decimal" },
  ] as const;

  return kind === "inventory"
    ? [...fields, { name: "quantity", label: "Остаток, шт.", inputMode: "numeric" } as const]
    : fields;
}

export function readCatalogInput(kind: CatalogKind, form: FormData): CatalogInput | { errors: FieldErrors } {
  const name = readFormText(form, "name");
  const rawPrice = readFormText(form, "price");
  const errors: FieldErrors = {};
  if (!name) errors.name = "Введите название.";

  const priceParts = /^([0-9]{1,10})(?:[.,]([0-9]{1,2}))?$/.exec(rawPrice);
  if (!priceParts) errors.price = "Введите цену до 9 999 999 999,99 ₽ с точностью до копеек.";

  const rawQuantity = readFormText(form, "quantity");
  const quantity = Number(rawQuantity);
  if (kind === "inventory" && (!/^[0-9]+$/.test(rawQuantity) || quantity > 2_147_483_647)) {
    errors.quantity = "Введите целое число от 0 до 2 147 483 647.";
  }

  if (!priceParts || Object.keys(errors).length) return { errors };
  const [, rubles, kopecks = ""] = priceParts;
  const price = `${rubles}.${kopecks.padEnd(2, "0")}`;
  return kind === "inventory"
    ? { kind, body: { name, price, quantity } }
    : { kind, body: { name, price } };
}
