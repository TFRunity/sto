"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type { FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { gateway, GatewayError } from "@/lib/api/client";
import { cabinetPath, readWebRole, sessionKey } from "./session";

const messages: Record<string, string> = {
  invalid_credentials: "Проверьте телефон и пароль.",
  phone_taken: "Этот телефон уже зарегистрирован.",
  validation_failed: "Проверьте данные формы.",
};

export function AuthForm({ register = false }: { register?: boolean }) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: async ({ name, phone, password }: { name: string; phone: string; password: string }) => {
      const { data } = register
        ? await gateway.POST("/register", { body: { name, phone, password } })
        : await gateway.POST("/login", { body: { phone, password } });
      const role = data?.token && readWebRole(data.token);
      if (!data?.token || !role) throw new Error("web_role");
      return { token: data.token, role };
    },
  });

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (mutation.isPending) return;
    const form = new FormData(event.currentTarget);
    mutation.mutate({
      name: String(form.get("name") ?? "").trim(),
      phone: String(form.get("phone") ?? "").trim(),
      password: String(form.get("password") ?? ""),
    }, {
      onSuccess({ token, role }) {
        queryClient.clear();
        sessionStorage.setItem(sessionKey, token);
        router.replace(cabinetPath[role]);
      },
    });
  }

  const error = mutation.error;
  const fields = error instanceof GatewayError ? error.fields : undefined;
  const phoneTaken = error instanceof GatewayError && error.code === "phone_taken";
  const message = error instanceof GatewayError
    ? messages[error.code ?? ""] ?? "Сервис временно недоступен. Попробуйте ещё раз."
    : error?.message === "web_role"
      ? "Веб-кабинет доступен клиенту и директору."
      : "Не удалось связаться с сервисом. Попробуйте ещё раз.";

  return (
    <main className="mx-auto max-w-sm space-y-4 rounded-lg border bg-card p-5 sm:p-6">
      <h1>{register ? "Регистрация клиента" : "Вход в СТО"}</h1>
      <form className="space-y-4" onSubmit={submit}>
        <fieldset className="space-y-4" disabled={mutation.isPending}>
          {register && <Field name="name" label="Имя" autoComplete="name" invalid={Boolean(fields?.name)} />}
          <Field name="phone" label="Телефон" type="tel" autoComplete="tel" invalid={phoneTaken || Boolean(fields?.phone)} />
          <Field name="password" label="Пароль" type="password" autoComplete={register ? "new-password" : "current-password"} invalid={Boolean(fields?.password)} />
          {error && <p role="alert" className="text-sm text-destructive">{message}</p>}
          <Button type="submit">{register ? "Зарегистрироваться" : "Войти"}</Button>
        </fieldset>
      </form>
      <Link className="text-accent-foreground underline" href={register ? "/login" : "/register"}>
        {register ? "Уже есть аккаунт" : "Создать аккаунт"}
      </Link>
    </main>
  );
}

function Field({ name, label, type = "text", autoComplete, invalid }: {
  name: string; label: string; type?: string; autoComplete: string; invalid: boolean;
}) {
  return (
    <div className="space-y-1">
      <label className="text-sm font-medium" htmlFor={name}>{label}</label>
      <Input id={name} name={name} type={type} autoComplete={autoComplete} required
        aria-invalid={invalid} aria-describedby={invalid ? `${name}-error` : undefined} />
      {invalid && <p id={`${name}-error`} className="text-sm text-destructive">Проверьте это поле.</p>}
    </div>
  );
}
