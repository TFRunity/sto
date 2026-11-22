"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";
import { SessionProvider } from "@/features/auth/session-provider";
import { GatewayError } from "@/lib/api/client";

export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(() => new QueryClient({
    defaultOptions: {
      queries: {
        retry: (failures, error) => {
          if (error instanceof GatewayError && error.status < 500) return false;
          return failures < 2;
        },
      },
    },
  }));

  return (
    <QueryClientProvider client={queryClient}>
      <SessionProvider>{children}</SessionProvider>
    </QueryClientProvider>
  );
}
