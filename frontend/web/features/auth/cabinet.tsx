"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useEffect, useState, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { cabinetPath, readWebRole, sessionExpired, sessionKey, type WebRole } from "./session";

export function Cabinet({ role, children }: { role: WebRole; children: ReactNode }) {
  const [ready, setReady] = useState(false);
  const router = useRouter();
  const queryClient = useQueryClient();

  useEffect(() => {
    function checkSession() {
      const token = sessionStorage.getItem(sessionKey);
      const currentRole = token ? readWebRole(token) : null;
      setReady(currentRole === role);
      if (currentRole === role) return;
      if (!currentRole) {
        sessionStorage.removeItem(sessionKey);
        queryClient.clear();
      }
      router.replace(currentRole ? cabinetPath[currentRole] : "/login");
    }

    checkSession();
    window.addEventListener(sessionExpired, checkSession);
    window.addEventListener("pageshow", checkSession);
    return () => {
      window.removeEventListener(sessionExpired, checkSession);
      window.removeEventListener("pageshow", checkSession);
    };
  }, [role, router, queryClient]);

  if (!ready) return <p role="status">Проверяем сессию…</p>;

  return (
    <main className="space-y-4">
      {children}
      <Button variant="outline" onClick={() => {
        sessionStorage.removeItem(sessionKey);
        queryClient.clear();
        setReady(false);
        router.replace("/login");
      }}>Выйти</Button>
    </main>
  );
}
