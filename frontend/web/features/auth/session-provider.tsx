"use client";

import { useQueryClient } from "@tanstack/react-query";
import {
  createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode,
} from "react";
import { createSessionGateway, register, signIn } from "./auth-api";
import { restoreSession, saveSession, watchSessionExpiry } from "./browser-session";

type Session = NonNullable<ReturnType<typeof restoreSession>>;

const SessionContext = createContext<{
  session: Session | null | undefined;
  gateway: ReturnType<typeof createSessionGateway>;
  signIn: typeof signIn;
  register: typeof register;
  signOut(): void;
} | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const current = useRef<Session | null>(null);
  const [session, setSession] = useState<Session | null>();

  const updateSession = useCallback((next: Session | null) => {
    saveSession(next);
    current.current = next;
    queryClient.clear();
    setSession(next);
  }, [queryClient]);

  useEffect(() => {
    const restored = restoreSession();
    current.current = restored;
    setSession(restored);
  }, []);

  useEffect(() => {
    if (!session) return;

    return watchSessionExpiry(session.expiresAt, () => {
      if (current.current === session) updateSession(null);
    });
  }, [session, updateSession]);

  const gateway = useMemo(() => createSessionGateway(
    () => {
      const active = current.current;
      return active && active.expiresAt > Date.now() ? active.token : undefined;
    },
    () => updateSession(null),
  ), [updateSession]);

  function startSession(next: Session) {
    updateSession(next);
    return next;
  }

  return (
    <SessionContext value={{
      session,
      gateway,
      signIn: (body, signal) => signIn(body, signal).then(startSession),
      register: (body, signal) => register(body, signal).then(startSession),
      signOut: () => updateSession(null),
    }}>
      {children}
    </SessionContext>
  );
}

export function useSession() {
  const value = useContext(SessionContext);
  if (!value) throw new Error("useSession requires SessionProvider");
  return value;
}
