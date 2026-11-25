"use client";

import { Button } from "@/components/ui/button";
import { useSession } from "@/common/session/view";
import { useDownloadCertificate } from "./controller";

export function CertificateDownload({ id }: { id: string }) {
  const { gateway } = useSession();
  const certificate = useDownloadCertificate(gateway, id);

  function download() {
    certificate.mutate(undefined, {
      onSuccess(pdf) {
        if (!pdf) return;
        const url = URL.createObjectURL(pdf);
        const anchor = document.createElement("a");
        anchor.href = url;
        anchor.download = `Справка-${id}.pdf`;
        document.body.append(anchor);
        anchor.click();
        anchor.remove();
        window.setTimeout(() => URL.revokeObjectURL(url), 1_000);
      },
    });
  }

  const unavailable = certificate.data === null;

  return (
    <div className="space-y-2">
      {unavailable && <p role="status">Справка пока недоступна.</p>}
      {certificate.isError && <p role="alert">Не удалось скачать справку. Попробуйте снова.</p>}
      <Button variant="outline" disabled={certificate.isPending} onClick={download}>
        {certificate.isPending ? "Загружаем справку…" : unavailable ? "Проверить справку снова" : "Скачать справку (PDF)"}
      </Button>
    </div>
  );
}
