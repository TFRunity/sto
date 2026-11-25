"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useState, type FormEvent } from "react";
import type { components } from "@shared/api/schema";
import { formatRubles } from "@shared/money/utils";
import { orderStageLabels } from "@shared/orders/labels";
import { useSession } from "@/common/session/view";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { GatewayError } from "@/lib/api/client";
import { useDirectorOrders } from "./controller";
import { ordersHref, readFilters, statusOptions, validateQuery, type OrderFilters } from "./form";

type Order = components["schemas"]["DirectorOrder"];

function label(labels: Readonly<Partial<Record<string, string>>>, value: string) {
  return Object.hasOwn(labels, value) ? labels[value] : value;
}

function SearchForm({ filters }: { filters: OrderFilters }) {
  const router = useRouter();
  const [error, setError] = useState(() => validateQuery(filters.q));

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const fields = new FormData(event.currentTarget);
    const next = readFilters(new URLSearchParams({
      q: String(fields.get("q") ?? ""),
      status: String(fields.get("status") ?? ""),
    }));
    const message = validateQuery(next.q);
    setError(message);
    if (!message) router.push(ordersHref(next));
  }

  return (
    <form onSubmit={submit} className="space-y-2">
      <div className="flex flex-wrap items-end gap-3">
        <label className="grid flex-1 gap-1 text-sm">
          Имя, телефон или госномер
          <Input name="q" defaultValue={filters.q} aria-invalid={Boolean(error)} aria-describedby={error ? "order-search-error" : undefined} />
        </label>
        <label className="grid gap-1 text-sm">
          Статус
          <select name="status" defaultValue={filters.status ?? ""} className="h-9 rounded-md border border-input bg-background px-3">
            <option value="">Все</option>
            {Object.entries(statusOptions).map(([value, title]) => <option key={value} value={value}>{title}</option>)}
          </select>
        </label>
        <Button type="submit">Найти</Button>
      </div>
      {error && <p id="order-search-error" role="alert" className="text-sm text-destructive">{error}</p>}
    </form>
  );
}

function OrderSummary({ order }: { order: Order }) {
  const fields = [
    ["ID заказа", order.id],
    ["Клиент", order.client_name],
    ["Телефон", order.client_phone],
    ["Госномер", order.vehicle_number],
    ["Инженер", order.engineer_name],
    ["Статус", label(statusOptions, order.status)],
    ["Этап", label(orderStageLabels, order.stage)],
    ["Сумма", formatRubles(order.total)],
  ];

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm" aria-label={`Сводка заказа ${order.id.slice(0, 8)}`}>Сводка</Button>
      </DialogTrigger>
      <DialogContent aria-describedby={undefined}>
        <DialogHeader><DialogTitle>Сводка заказа</DialogTitle></DialogHeader>
        <dl className="grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm">
          {fields.map(([name, value]) => (
            <div key={name} className="contents">
              <dt className="text-muted-foreground">{name}</dt>
              <dd className="wrap-anywhere">{value || "—"}</dd>
            </div>
          ))}
        </dl>
      </DialogContent>
    </Dialog>
  );
}

function ResultsTable({ orders }: { orders: Order[] }) {
  return (
    <div className="rounded-lg border bg-card">
      <Table aria-label="Заказы">
        <TableHeader>
          <TableRow>
            {["Госномер", "Клиент", "Инженер", "Сумма", "Статус", "Этап"].map((title) => (
              <TableHead key={title} scope="col">{title}</TableHead>
            ))}
            <TableHead scope="col"><span className="sr-only">Действия</span></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {orders.map((order) => (
            <TableRow key={order.id}>
              <TableCell>{order.vehicle_number || "—"}</TableCell>
              <TableCell>{order.client_name || "—"}</TableCell>
              <TableCell>{order.engineer_name || "—"}</TableCell>
              <TableCell>{formatRubles(order.total)}</TableCell>
              <TableCell>{label(statusOptions, order.status)}</TableCell>
              <TableCell>{label(orderStageLabels, order.stage)}</TableCell>
              <TableCell>
                <div className="flex flex-wrap items-center gap-2">
                  <OrderSummary order={order} />
                  <Button asChild variant="outline" size="sm">
                    <Link href={`/director/orders/${order.id}`} aria-label={`Открыть заказ ${order.id.slice(0, 8)}`}>Детали</Link>
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

export function DirectorOrderSearch() {
  const filters = readFilters(useSearchParams());
  const { gateway } = useSession();
  const valid = !validateQuery(filters.q);
  const results = useDirectorOrders(gateway, filters, valid);

  return (
    <div className="space-y-5">
      <h1>Все заказы</h1>
      <SearchForm key={ordersHref({ ...filters, page: 1 })} filters={filters} />
      {valid && results.isPending && <p role="status">Загружаем заказы…</p>}
      {valid && results.isError && (
        results.error instanceof GatewayError && results.error.code === "search_too_broad"
          ? <p role="alert">Слишком много совпадений. Уточните имя, телефон или госномер.</p>
          : <div className="space-y-2">
              <p role="alert">Не удалось загрузить заказы.</p>
              <Button variant="outline" disabled={results.isFetching} onClick={() => void results.refetch()}>Повторить</Button>
            </div>
      )}
      {valid && results.isSuccess && (
        <>
          {results.data.orders.length ? <ResultsTable orders={results.data.orders} /> : <p>Заказы не найдены.</p>}
          <nav aria-label="Страницы заказов" className="flex flex-wrap items-center gap-3">
            {results.data.page > 1 && (
              <Button asChild variant="outline">
                <Link href={ordersHref({ ...filters, page: results.data.page - 1 })}>Назад</Link>
              </Button>
            )}
            <span>Страница {results.data.page}</span>
            {results.data.has_next && (
              <Button asChild variant="outline">
                <Link href={ordersHref({ ...filters, page: results.data.page + 1 })}>Вперёд</Link>
              </Button>
            )}
          </nav>
        </>
      )}
    </div>
  );
}
