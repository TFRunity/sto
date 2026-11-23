import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { GatewayError } from "@/lib/api/client";

export function ReadState({ pending, error, retry, children }: {
  pending: boolean; error: Error | null; retry: () => void; children?: ReactNode;
}) {
  if (pending) return <p role="status" className="text-muted-foreground">Загружаем…</p>;
  if (!error) return children;
  if (error instanceof GatewayError && [403, 404].includes(error.status)) {
    return <p role="alert">Объект недоступен или удалён.</p>;
  }
  return (
    <div className="space-y-2">
      <p role="alert">Не удалось загрузить данные.</p>
      <Button variant="outline" onClick={retry}>Повторить</Button>
    </div>
  );
}

export const panel = "rounded-xl border bg-card p-4 sm:p-5";
export const link = "font-medium text-accent-foreground underline-offset-4 hover:underline";
