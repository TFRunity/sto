import type { Metadata } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import { Providers } from "./providers";
import "./globals.css";

export const metadata: Metadata = {
  title: "СТО",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="ru">
      <body className="mx-auto max-w-5xl px-4 py-6 sm:px-8">
        <header className="mb-6 border-b border-border pb-4">
          <nav aria-label="Кабинеты">
            <ul className="flex flex-wrap gap-x-6 gap-y-2">
              <li><Link href="/login">Вход</Link></li>
              <li>
                <Link className="rounded-sm underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-ring" href="/customer">
                  Кабинет клиента
                </Link>
              </li>
              <li>
                <Link className="rounded-sm underline-offset-4 hover:underline focus-visible:outline-2 focus-visible:outline-ring" href="/director">
                  Кабинет директора
                </Link>
              </li>
            </ul>
          </nav>
        </header>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
