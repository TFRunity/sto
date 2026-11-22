import type { Metadata } from "next";
import type { ReactNode } from "react";
import { Providers } from "./providers";
import "./globals.css";

export const metadata: Metadata = {
  title: "СТО",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="ru">
      <body>
        {/* Radix блокирует прокрутку на body и временно меняет его отступы. */}
        <div className="mx-auto max-w-5xl px-4 py-6 sm:px-8">
          <Providers>{children}</Providers>
        </div>
      </body>
    </html>
  );
}
