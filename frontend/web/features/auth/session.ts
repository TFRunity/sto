export type WebRole = "client" | "director";
export const cabinetPath = { client: "/customer", director: "/director" } as const;

export function readWebSession(token: string) {
  try {
    const payload = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const { role, exp } = JSON.parse(atob(payload));
    if (typeof exp !== "number" || !Number.isFinite(exp) || exp * 1000 <= Date.now()) return null;
    if (role !== "client" && role !== "director") return null;

    return { token, role: role as WebRole, expiresAt: exp * 1000 };
  } catch {
    return null;
  }
}
