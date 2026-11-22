import type { ReactNode } from "react";
import { Cabinet } from "@/features/auth/cabinet";

export default function DirectorLayout({ children }: { children: ReactNode }) {
  return <Cabinet role="director">{children}</Cabinet>;
}
