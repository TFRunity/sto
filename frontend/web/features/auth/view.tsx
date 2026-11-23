"use client";

import { Wrench } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Fragment, useEffect, useRef, useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { cabinetPath, type WebRole } from "@/common/session/session";
import { useSession } from "@/common/session/view";
import { register as registerClient, signIn } from "./controller";
import { authFields, describeAuthError, readAuthForm, type AuthErrors } from "./form";

export function AuthForm({ register = false }: { register?: boolean }) {
  const router = useRouter();
  const { pending, errors, message, submit, clearField } = useAuthForm(register, (role) => {
    router.replace(cabinetPath[role]);
  });
  const [showPassword, setShowPassword] = useState(false);
  const fields = authFields({
    register, errors, clearField, showPassword,
    togglePassword: () => setShowPassword((visible) => !visible),
  });

  return (
    <main className="mx-auto grid min-h-[calc(100svh-3rem)] max-w-4xl items-center">
      <div className="grid overflow-hidden rounded-2xl border bg-card shadow-sm md:grid-cols-[0.85fr_1fr]">
        <div className="flex gap-4 bg-[#172b38] p-6 text-white md:flex-col md:justify-between md:p-9">
          <div className="flex items-center gap-3">
            <span className="grid size-11 place-items-center rounded-xl bg-white/10" aria-hidden="true">
              <Wrench className="size-5" />
            </span>
            <span className="text-xl font-semibold">СТО</span>
          </div>
          <p className="hidden max-w-xs text-3xl leading-tight font-semibold md:block">
            Автомобили, заявки и история сервиса
          </p>
        </div>

        <div className="space-y-6 p-5 sm:p-8 md:p-10">
          <header className="space-y-2">
            <h1>{register ? "Регистрация клиента" : "Вход в личный кабинет"}</h1>
            <p className="text-muted-foreground">
              {register ? "Укажите данные, чтобы записаться на сервис." : "Введите телефон и пароль."}
            </p>
          </header>

          <form noValidate onSubmit={submit} aria-busy={pending}>
            <fieldset disabled={pending} className="space-y-4">
              {fields.map(({ name, render }) => <Fragment key={name}>{render()}</Fragment>)}

              {message && <p role="alert" className="text-sm text-destructive">{message}</p>}
              <Button type="submit" className="h-12 w-full bg-[#172b38] text-base hover:bg-[#234558]">
                {pending ? "Отправляем…" : register ? "Зарегистрироваться" : "Войти"}
              </Button>
            </fieldset>
          </form>

          <p className="border-t pt-4 text-sm text-muted-foreground">
            {register ? "Уже зарегистрированы? " : "Впервые в СТО? "}
            <Link className="font-semibold text-accent-foreground underline underline-offset-4"
              href={register ? "/login" : "/register"}>
              {register ? "Войти" : "Зарегистрироваться"}
            </Link>
          </p>
        </div>
      </div>
    </main>
  );
}

function useAuthForm(register: boolean, onSignedIn: (role: WebRole) => void) {
  const { startSession } = useSession();
  const request = useRef<AbortController | null>(null);
  const [pending, setPending] = useState(false);
  const [errors, setErrors] = useState<AuthErrors>({});
  const [message, setMessage] = useState<string>();

  // Уход со страницы отменяет запрос, чтобы запоздалый ответ не открыл кабинет.
  useEffect(() => () => request.current?.abort(), []);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (request.current) return;

    const result = readAuthForm(new FormData(event.currentTarget), register);
    setMessage(undefined);

    if (!result.body) {
      setErrors(result.errors);
      return;
    }

    const controller = new AbortController();
    const { name, phone, password } = result.body;
    request.current = controller;
    setErrors({});
    setPending(true);

    const response = register
      ? registerClient({ name, phone, password }, controller.signal)
      : signIn({ phone, password }, controller.signal);

    void response.then((session) => {
      if (controller.signal.aborted) return;
      startSession(session);
      onSignedIn(session.role);
    }).catch((error: unknown) => {
      if (controller.signal.aborted) return;

      const feedback = describeAuthError(error);
      setErrors(feedback.fields ?? {});
      setMessage(feedback.message);
    }).finally(() => {
      request.current = null;
      if (!controller.signal.aborted) setPending(false);
    });
  }

  function clearField(name: keyof AuthErrors) {
    setErrors((current) => current[name] ? { ...current, [name]: undefined } : current);
  }

  return { pending, errors, message, submit, clearField };
}
