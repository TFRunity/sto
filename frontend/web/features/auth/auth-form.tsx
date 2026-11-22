"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { GatewayError } from "@/lib/api/client";
import { cabinetPath } from "./session";
import { useSession, WebRoleError } from "./session-provider";

const fieldMessages: Record<string, string> = {
  required: "Заполните это поле.",
  too_long: "Пароль должен занимать не более 72 байт.",
  format: "Проверьте формат значения.",
  minLength: "Значение слишком короткое.",
  maxLength: "Значение слишком длинное.",
};

function fieldError(error: GatewayError | null, name: string) {
  if (name === "phone" && error?.code === "phone_taken") {
    return "Этот телефон уже зарегистрирован.";
  }
  const reason = error?.fields?.[name];
  return reason ? fieldMessages[reason] ?? "Проверьте значение поля." : undefined;
}

function formError(error: unknown) {
  if (error instanceof WebRoleError) return "Веб-кабинет доступен клиенту и директору.";
  if (error instanceof GatewayError) {
    if (error.code === "invalid_credentials") return "Проверьте телефон и пароль.";
    if (error.code === "phone_taken") return null;
    if (error.code === "validation_failed") {
      return ["name", "phone", "password"].some((name) => error.fields?.[name])
        ? null
        : "Проверьте данные формы.";
    }
  }
  return "Не удалось связаться с сервисом. Попробуйте ещё раз.";
}

export function AuthForm({ register = false }: { register?: boolean }) {
  const router = useRouter();
  const auth = useSession();
  const request = useRef<AbortController | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<unknown>(null);

  useEffect(() => () => request.current?.abort(), []);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (request.current) return;

    const form = new FormData(event.currentTarget);
    const phone = String(form.get("phone") ?? "").trim();
    const password = String(form.get("password") ?? "");
    const controller = new AbortController();
    request.current = controller;
    setPending(true);
    setError(null);

    try {
      const session = register
        ? await auth.register({ name: String(form.get("name") ?? "").trim(), phone, password }, controller.signal)
        : await auth.signIn({ phone, password }, controller.signal);
      if (!controller.signal.aborted) router.replace(cabinetPath[session.role]);
    } catch (cause) {
      if (!controller.signal.aborted) setError(cause);
    } finally {
      request.current = null;
      if (!controller.signal.aborted) setPending(false);
    }
  }

  const gatewayError = error instanceof GatewayError ? error : null;
  const message = error ? formError(error) : null;

  return (
    <main className="mx-auto max-w-sm space-y-4 rounded-lg border bg-card p-5 sm:p-6">
      <h1>{register ? "Регистрация клиента" : "Вход в СТО"}</h1>
      <form className="space-y-4" onSubmit={submit} aria-busy={pending}>
        <fieldset className="space-y-4" disabled={pending}>
          {register && <Field name="name" label="Имя" autoComplete="name" error={fieldError(gatewayError, "name")} />}
          <Field name="phone" label="Телефон" type="tel" autoComplete="tel" error={fieldError(gatewayError, "phone")} />
          <Field name="password" label="Пароль" type="password" autoComplete={register ? "new-password" : "current-password"} error={fieldError(gatewayError, "password")} />
          {message && <p role="alert" className="text-sm text-destructive">{message}</p>}
          <Button type="submit">
            {pending ? "Отправляем…" : register ? "Зарегистрироваться" : "Войти"}
          </Button>
        </fieldset>
      </form>
      <Link className="text-accent-foreground underline" href={register ? "/login" : "/register"}>
        {register ? "Уже есть аккаунт" : "Создать аккаунт"}
      </Link>
    </main>
  );
}

function Field({ name, label, type = "text", autoComplete, error }: {
  name: string; label: string; type?: string; autoComplete: string; error?: string;
}) {
  return (
    <div className="space-y-1">
      <label className="text-sm font-medium" htmlFor={name}>{label}</label>
      <Input id={name} name={name} type={type} autoComplete={autoComplete} required
        aria-invalid={Boolean(error)} aria-describedby={error ? `${name}-error` : undefined} />
      {error && <p role="alert" id={`${name}-error`} className="text-sm text-destructive">{error}</p>}
    </div>
  );
}
