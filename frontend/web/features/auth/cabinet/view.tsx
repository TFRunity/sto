"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { cabinetPath, type WebRole } from "@/common/session/session";
import { useSession } from "@/common/session/view";

const labels = { client: "Кабинет клиента", director: "Кабинет директора" };

export function Cabinet({ role, children }: { role: WebRole; children: ReactNode }) {
  const { session, signOut } = useSession();
  const router = useRouter();

  useEffect(() => {
    if (session === undefined || session?.role === role) return;
    router.replace(session ? cabinetPath[session.role] : "/login");
  }, [role, router, session]);

  if (session?.role !== role) return <p role="status">Проверяем сессию…</p>;

  return (
    <>
      <header className="mb-6 flex flex-wrap items-center justify-between gap-3 border-b pb-4">
        <nav aria-label={labels[role]} className="flex flex-wrap items-center gap-4">
          <Link className="font-medium underline-offset-4 hover:underline" href={cabinetPath[role]}>
            {role === "client" ? "Мои заказы" : "Дашборд"}
          </Link>
          {role === "client" && (
            <Link className="font-medium underline-offset-4 hover:underline" href="/customer/certificates">
              Справки
            </Link>
          )}
          {role === "director" && (
            <>
              <Link className="font-medium underline-offset-4 hover:underline" href="/director/orders">
                Заказы
              </Link>
              <Link className="font-medium underline-offset-4 hover:underline" href="/director/catalog">
                Каталог
              </Link>
            </>
          )}
          <Link className="font-medium underline-offset-4 hover:underline" href={`${cabinetPath[role]}/profile`}>
            Профиль
          </Link>
        </nav>
        <Button variant="outline" onClick={signOut}>Выйти</Button>
      </header>
      <main>{children}</main>
    </>
  );
}
