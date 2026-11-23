"use client";

import { useQueryClient } from "@tanstack/react-query";
import {
  createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode,
} from "react";
import { createSessionGateway } from "./controller";
import { restoreSession, saveSession, watchSessionExpiry, type Session } from "./session";

const SessionContext = createContext<{
  session: Session | null | undefined;
  gateway: ReturnType<typeof createSessionGateway>;
  startSession(session: Session): void;
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

  return (
    <SessionContext value={{
      session,
      gateway,
      startSession: updateSession,
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
