import type { components } from "../../../common/api/schema";
import { gatewayRequest } from "../../common/api/controller";

type LoginRequest = components["schemas"]["LoginRequest"];
type TokenResponse = components["schemas"]["TokenResponse"];

export function loginEngineer(body: LoginRequest) {
  return gatewayRequest("/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  }).then((response) => response.json() as Promise<TokenResponse>).then(({ token }) => {
    if (!token) throw new Error("Missing access token");
    return token;
  });
}
