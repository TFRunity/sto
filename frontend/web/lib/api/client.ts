import createClient from "openapi-fetch";
import type { components, paths } from "@shared/api/schema";

type ErrorBody = components["schemas"]["ErrorBody"];

function readErrorBody(value: unknown): ErrorBody | undefined {
  if (typeof value !== "object" || value === null || !("error" in value)) {
    return;
  }

  const body = value.error;
  if (
    typeof body !== "object" || body === null ||
    !("code" in body) || typeof body.code !== "string" ||
    !("message" in body) || typeof body.message !== "string"
  ) {
    return;
  }

  let fields: ErrorBody["fields"];
  if (
    "fields" in body &&
    typeof body.fields === "object" &&
    body.fields !== null &&
    !Array.isArray(body.fields)
  ) {
    fields = Object.fromEntries(
      Object.entries(body.fields).filter((entry): entry is [string, string] => typeof entry[1] === "string"),
    );
  }

  return { code: body.code, message: body.message, fields };
}

export class GatewayError extends Error {
  readonly status: number;
  readonly code: string | undefined;
  readonly fields: ErrorBody["fields"];

  constructor(status: number, body?: ErrorBody) {
    super(body?.message ?? `Gateway returned HTTP ${status}`);
    this.name = "GatewayError";
    this.status = status;
    this.code = body?.code;
    this.fields = body?.fields;
  }
}

export function createGatewayClient(baseUrl = "/api", fetcher?: typeof fetch) {
  const client = createClient<paths>({ baseUrl, fetch: fetcher });
  client.use({
    async onResponse({ response }) {
      if (response.ok) return;

      const payload: unknown = await response.json().catch(() => undefined);
      throw new GatewayError(response.status, readErrorBody(payload));
    },
  });

  return client;
}

export type GatewayClient = ReturnType<typeof createGatewayClient>;
