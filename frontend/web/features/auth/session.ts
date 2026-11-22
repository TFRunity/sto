export type WebRole = "client" | "director";
export const cabinetPath = { client: "/customer", director: "/director" } as const;
export const sessionKey = "sto.web.session";
export const sessionExpired = "sto:session-expired";

export function readWebRole(token: string): WebRole | null {
  try {
    const payload = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const { role, exp } = JSON.parse(atob(payload));
    if (typeof exp !== "number" || exp * 1000 <= Date.now()) return null;
    return role === "client" || role === "director" ? role : null;
  } catch {
    return null;
  }
}
