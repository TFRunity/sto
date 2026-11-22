"use client";

import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { describeAuthError, readAuthForm, type AuthErrors } from "./auth-validation";
import { cabinetPath } from "./session";
import { useSession } from "./session-provider";

export function useAuthForm(register: boolean) {
  const router = useRouter();
  const auth = useSession();
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
      ? auth.register({ name, phone, password }, controller.signal)
      : auth.signIn({ phone, password }, controller.signal);

    void response.then((session) => {
      if (!controller.signal.aborted) router.replace(cabinetPath[session.role]);
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
