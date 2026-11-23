export type WebRole = "client" | "director";
export const cabinetPath = { client: "/customer", director: "/director" } as const;

export function readWebSession(token: string) {
  try {
    // Здесь читаем роль для навигации. Подпись и право доступа проверяет Gateway.
    const payload = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const { role, exp } = JSON.parse(atob(payload));
    if (typeof exp !== "number" || !Number.isFinite(exp) || exp * 1000 <= Date.now()) return null;
    if (role !== "client" && role !== "director") return null;

    return { token, role: role as WebRole, expiresAt: exp * 1000 };
  } catch {
    return null;
  }
}

export class WebRoleError extends Error {}

const storageKey = "sto.web.session";
export type Session = NonNullable<ReturnType<typeof readWebSession>>;

export function restoreSession() {
  const token = sessionStorage.getItem(storageKey);
  const session = token ? readWebSession(token) : null;
  if (token && !session) sessionStorage.removeItem(storageKey);
  return session;
}

export function saveSession(session: Session | null) {
  if (session) sessionStorage.setItem(storageKey, session.token);
  else sessionStorage.removeItem(storageKey);
}

export function acceptSession(token: string | undefined, signal: AbortSignal) {
  signal.throwIfAborted();
  const session = token ? readWebSession(token) : null;
  if (!session) throw new WebRoleError();
  return session;
}

export function watchSessionExpiry(expiresAt: number, onExpire: () => void) {
  function checkExpiry() {
    if (Date.now() >= expiresAt) onExpire();
  }

  // Браузер может отложить таймер в фоновой вкладке. Перепроверяем срок JWT
  // при возвращении на страницу и при восстановлении страницы из bfcache.
  const timer = window.setTimeout(checkExpiry, Math.max(0, expiresAt - Date.now()));
  window.addEventListener("pageshow", checkExpiry);
  document.addEventListener("visibilitychange", checkExpiry);

  return () => {
    window.clearTimeout(timer);
    window.removeEventListener("pageshow", checkExpiry);
    document.removeEventListener("visibilitychange", checkExpiry);
  };
}
