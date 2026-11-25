import { useMutation } from "@tanstack/react-query";
import { GatewayError, type GatewayClient } from "@/lib/api/client";

export function useDownloadCertificate(gateway: GatewayClient, id: string) {
  return useMutation({
    mutationFn: async () => {
      try {
        const { data } = await gateway.GET("/orders/{id}/certificate", {
          params: { path: { id } },
          headers: { Accept: "application/pdf" },
          parseAs: "blob",
        });
        if (!data?.size) throw new Error("Certificate PDF is empty");
        return data;
      } catch (error) {
        if (error instanceof GatewayError && error.status === 404) return null;
        throw error;
      }
    },
  });
}
