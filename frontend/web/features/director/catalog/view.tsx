"use client";

import Link from "next/link";
import { useId, useState, type FormEvent } from "react";
import { formatRubles } from "@shared/money/utils";
import type { FieldErrors } from "@/common/hooks/use-form";
import { useSession } from "@/common/session/view";
import { Button } from "@/components/ui/button";
import {
  Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { GatewayError } from "@/lib/api/client";
import { useCatalog, useCatalogChange, type CatalogEntry } from "./controller";
import { catalogFields, readCatalogInput, type CatalogKind } from "./form";

function catalogError(error: Error, fallback: string) {
  if (error instanceof GatewayError) {
    if (error.status === 404) return "Позиция больше недоступна. Обновите каталог.";
    if (error.code === "validation_failed") return "Проверьте данные позиции.";
  }
  return fallback;
}

function CatalogDialog({ kind, entry, deleting = false }: {
  kind: CatalogKind;
  entry?: CatalogEntry;
  deleting?: boolean;
}) {
  const { gateway } = useSession();
  const mutation = useCatalogChange(gateway);
  const [open, setOpen] = useState(false);
  const [errors, setErrors] = useState<FieldErrors>({});
  const fieldId = useId();
  const addLabel = kind === "inventory" ? "Добавить товар" : "Добавить услугу";
  const action = deleting ? "Удалить" : entry ? "Изменить" : addLabel;
  const defaults = {
    name: entry?.name,
    price: entry?.price,
    quantity: entry && "quantity" in entry ? entry.quantity : undefined,
  };

  function close(next: boolean) {
    if (mutation.isPending) return;
    if (next) {
      setErrors({});
      mutation.reset();
    }
    setOpen(next);
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (mutation.isPending) return;
    const onSuccess = () => setOpen(false);
    if (deleting && entry) {
      mutation.mutate({ action: "delete", kind, id: entry.id }, { onSuccess });
      return;
    }

    const input = readCatalogInput(kind, new FormData(event.currentTarget));
    mutation.reset();
    if ("errors" in input) {
      setErrors(input.errors);
      return;
    }
    setErrors({});
    mutation.mutate({ action: "save", input, id: entry?.id }, { onSuccess });
  }

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogTrigger asChild>
        <Button variant={entry ? "outline" : "default"} size={entry ? "sm" : "default"}
          aria-label={entry ? `${action} «${entry.name}»` : undefined}>
          {action}
        </Button>
      </DialogTrigger>
      <DialogContent showCloseButton={!mutation.isPending} aria-describedby={deleting ? `${fieldId}-description` : undefined}>
        <DialogHeader>
          <DialogTitle>{deleting ? "Удалить позицию?" : entry ? "Изменить позицию" : addLabel}</DialogTitle>
          {deleting && (
            <DialogDescription id={`${fieldId}-description`}>
              «{entry?.name}» исчезнет из каталога и останется в ранее оформленных заказах.
            </DialogDescription>
          )}
        </DialogHeader>
        <form onSubmit={submit} aria-busy={mutation.isPending} className="space-y-4">
          {!deleting && (
            <fieldset disabled={mutation.isPending} className="space-y-3">
              {catalogFields(kind).map(({ name, label, inputMode }) => {
                const id = `${fieldId}-${name}`;
                return (
                  <div key={name} className="space-y-1">
                    <label htmlFor={id} className="text-sm font-medium">{label}</label>
                    <Input id={id} name={name} inputMode={inputMode} defaultValue={defaults[name]}
                      aria-invalid={Boolean(errors[name])} aria-describedby={errors[name] ? `${id}-error` : undefined} />
                    {errors[name] && <p id={`${id}-error`} role="alert" className="text-sm text-destructive">{errors[name]}</p>}
                  </div>
                );
              })}
            </fieldset>
          )}
          {mutation.error && <p role="alert" className="text-sm text-destructive">
            {catalogError(mutation.error, deleting ? "Не удалось удалить позицию." : "Не удалось сохранить позицию.")}
          </p>}
          <DialogFooter>
            <DialogClose asChild><Button type="button" variant="outline" disabled={mutation.isPending}>Отмена</Button></DialogClose>
            <Button type="submit" variant={deleting ? "destructive" : "default"} disabled={mutation.isPending}>
              {deleting ? "Удалить" : "Сохранить"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export function DirectorCatalog({ kind }: { kind: CatalogKind }) {
  const { gateway } = useSession();
  const catalog = useCatalog(gateway, kind);
  const inventory = kind === "inventory";

  return (
    <div className="space-y-5">
      <h1>Каталог</h1>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <nav aria-label="Разделы каталога" className="flex flex-wrap gap-2">
          <Button asChild variant={inventory ? "secondary" : "ghost"}>
            <Link href="/director/catalog" aria-current={inventory ? "page" : undefined}>Товары</Link>
          </Button>
          <Button asChild variant={inventory ? "ghost" : "secondary"}>
            <Link href="/director/catalog/services" aria-current={inventory ? undefined : "page"}>Услуги</Link>
          </Button>
        </nav>
        <CatalogDialog key={kind} kind={kind} />
      </div>
      {catalog.isPending && <p role="status">Загружаем каталог…</p>}
      {catalog.isError && (
        <div className="space-y-2">
          <p role="alert">Не удалось загрузить каталог.</p>
          <Button variant="outline" disabled={catalog.isFetching} onClick={() => void catalog.refetch()}>Повторить</Button>
        </div>
      )}
      {catalog.isSuccess && (catalog.data.length ? (
        <div className="rounded-lg border bg-card">
          <Table aria-label={inventory ? "Товары" : "Услуги"}>
            <TableHeader>
              <TableRow>
                <TableHead scope="col">Название</TableHead>
                <TableHead scope="col">Цена</TableHead>
                {inventory && <TableHead scope="col">Остаток, шт.</TableHead>}
                <TableHead scope="col"><span className="sr-only">Действия</span></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {catalog.data.map((entry) => (
                <TableRow key={entry.id}>
                  <TableCell className="whitespace-normal">{entry.name}</TableCell>
                  <TableCell>{formatRubles(entry.price)}</TableCell>
                  {inventory && <TableCell>{"quantity" in entry ? entry.quantity : "—"}</TableCell>}
                  <TableCell>
                    <div className="flex justify-end gap-2">
                      <CatalogDialog kind={kind} entry={entry} />
                      <CatalogDialog kind={kind} entry={entry} deleting />
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : <p>{inventory ? "Товаров пока нет." : "Услуг пока нет."}</p>)}
    </div>
  );
}
