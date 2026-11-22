"use client";

import { useQueryClient } from "@tanstack/react-query";
import {
  createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode,
} from "react";
import { createGatewayClient } from "@/lib/api/client";
import type { components } from "@/lib/api/schema";
import { readWebSession } from "./session";

type Schemas = components["schemas"];
type Session = NonNullable<ReturnType<typeof readWebSession>>;

const storageKey = "sto.web.session";
const publicGateway = createGatewayClient();

export class WebRoleError extends Error {}

const SessionContext = createContext<{
  session: Session | null | undefined;
  gateway: ReturnType<typeof createGatewayClient>;
  signIn(body: Schemas["LoginRequest"], signal: AbortSignal): Promise<Session>;
  register(body: Schemas["RegisterRequest"], signal: AbortSignal): Promise<Session>;
  signOut(): void;
} | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const current = useRef<Session | null>(null);
  const [session, setSession] = useState<Session | null>();

  const updateSession = useCallback((next: Session | null) => {
    if (next) sessionStorage.setItem(storageKey, next.token);
    else sessionStorage.removeItem(storageKey);
    current.current = next;
    queryClient.clear();
    setSession(next);
  }, [queryClient]);

  useEffect(() => {
    const token = sessionStorage.getItem(storageKey);
    const restored = token ? readWebSession(token) : null;
    if (token && !restored) sessionStorage.removeItem(storageKey);
    current.current = restored;
    setSession(restored);
  }, []);

  useEffect(() => {
    if (!session) return;
    const { expiresAt } = session;
    function expireIfNeeded() {
      if (current.current === session && Date.now() >= expiresAt) {
        updateSession(null);
      }
    }
    const timer = window.setTimeout(expireIfNeeded, Math.max(0, expiresAt - Date.now()));
    window.addEventListener("pageshow", expireIfNeeded);
    document.addEventListener("visibilitychange", expireIfNeeded);
    return () => {
      window.clearTimeout(timer);
      window.removeEventListener("pageshow", expireIfNeeded);
      document.removeEventListener("visibilitychange", expireIfNeeded);
    };
  }, [session, updateSession]);

  const gateway = useMemo(() => {
    const client = createGatewayClient();
    client.use({
      onRequest({ request }) {
        const active = current.current;
        if (active && active.expiresAt > Date.now()) {
          request.headers.set("Authorization", `Bearer ${active.token}`);
        }
      },
      onResponse({ request, response }) {
        const active = current.current;
        if (response.status === 401 && active &&
          request.headers.get("Authorization") === `Bearer ${active.token}`) {
          updateSession(null);
        }
      },
    });
    return client;
  }, [updateSession]);

  function accept(token: string | undefined) {
    const next = token ? readWebSession(token) : null;
    if (!next) throw new WebRoleError();
    updateSession(next);
    return next;
  }

  return (
    <SessionContext.Provider value={{
      session,
      gateway,
      async signIn(body, signal) {
        const { data } = await publicGateway.POST("/login", { body, signal });
        signal.throwIfAborted();
        return accept(data?.token);
      },
      async register(body, signal) {
        const { data } = await publicGateway.POST("/register", { body, signal });
        signal.throwIfAborted();
        return accept(data?.token);
      },
      signOut: () => updateSession(null),
    }}>
      {children}
    </SessionContext.Provider>
  );
}

export function useSession() {
  const value = useContext(SessionContext);
  if (!value) throw new Error("useSession requires SessionProvider");
  return value;
}
