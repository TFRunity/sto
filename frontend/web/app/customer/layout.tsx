import type { ReactNode } from "react";
import { Cabinet } from "@/features/auth/cabinet";

export default function CustomerLayout({ children }: { children: ReactNode }) {
  return <Cabinet role="client">{children}</Cabinet>;
}
