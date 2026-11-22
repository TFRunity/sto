"use client";

import { Eye, EyeOff, Wrench } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { IMaskInput } from "react-imask";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { preparePhoneInput, russianPhoneMask, russianPhonePlaceholder } from "@/lib/forms/masks";
import { useAuthForm } from "./use-auth-form";

const inputClass = "h-12 w-full rounded-lg border border-input bg-white px-3.5 text-base shadow-xs";
const maskFocusClass = "focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/30 focus-visible:outline-none aria-invalid:border-destructive";

export function AuthForm({ register = false }: { register?: boolean }) {
  const { pending, errors, message, submit, clearField } = useAuthForm(register);
  const [showPassword, setShowPassword] = useState(false);

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
              {register && (
                <div className="space-y-1.5">
                  <label className="font-medium" htmlFor="name">Имя</label>
                  <Input id="name" name="name" required autoComplete="name" className={inputClass}
                    placeholder="Иван Иванов" aria-invalid={Boolean(errors.name)}
                    aria-describedby={errors.name ? "name-error" : undefined}
                    onChange={() => clearField("name")} />
                  <FieldFeedback name="name" error={errors.name} />
                </div>
              )}

              <div className="space-y-1.5">
                <label className="font-medium" htmlFor="phone">Телефон</label>
                <IMaskInput id="phone" name="phone" type="text" inputMode="tel" autoComplete="tel"
                  required mask={russianPhoneMask} prepare={preparePhoneInput}
                  placeholder={russianPhonePlaceholder} className={`${inputClass} ${maskFocusClass}`}
                  aria-invalid={Boolean(errors.phone)} aria-describedby={errors.phone ? "phone-error" : undefined}
                  onAccept={() => clearField("phone")} />
                <FieldFeedback name="phone" error={errors.phone} />
              </div>

              <div className="space-y-1.5">
                <label className="font-medium" htmlFor="password">Пароль</label>
                <div className="relative">
                  <Input id="password" name="password" type={showPassword ? "text" : "password"}
                    className={`${inputClass} pr-12`} required
                    autoComplete={register ? "new-password" : "current-password"}
                    aria-invalid={Boolean(errors.password)}
                    aria-describedby={errors.password ? "password-error" : register ? "password-hint" : undefined}
                    onChange={() => clearField("password")} />
                  <Button type="button" variant="ghost" size="icon" className="absolute top-1.5 right-1.5"
                    aria-label={showPassword ? "Скрыть пароль" : "Показать пароль"}
                    onClick={() => setShowPassword(!showPassword)}>
                    {showPassword ? <EyeOff aria-hidden="true" /> : <Eye aria-hidden="true" />}
                  </Button>
                </div>
                <FieldFeedback name="password" error={errors.password} />
                {register && !errors.password && (
                  <p id="password-hint" className="text-sm text-muted-foreground">Не менее 8 символов.</p>
                )}
              </div>

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

function FieldFeedback({ name, error }: { name: string; error?: string }) {
  return error
    ? <p role="alert" id={`${name}-error`} className="text-sm text-destructive">{error}</p>
    : null;
}
