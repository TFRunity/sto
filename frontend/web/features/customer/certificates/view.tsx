"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { certificateStatusLabels } from "@shared/orders/certificate";
import { useSession } from "@/common/session/view";
import { CertificateDownload } from "@/common/certificates/view";
import { orderPath } from "../shared/routes";
import { ReadState, link, panel } from "../shared/view";
import { certificatesQuery, type Certificate } from "./controller";

const dateFormat = new Intl.DateTimeFormat("ru-RU", { dateStyle: "medium", timeZone: "Europe/Moscow" });

export function CertificatesScreen() {
  const { gateway } = useSession();
  const certificates = useQuery(certificatesQuery(gateway));

  return (
    <section className="mx-auto max-w-2xl space-y-4">
      <h1>Справки</h1>
      <ReadState pending={certificates.isPending} error={certificates.error} retry={() => void certificates.refetch()}>
        {certificates.data?.length ? (
          <ul className="space-y-3">
            {certificates.data.map((certificate) => <CertificateRow key={certificate.id} certificate={certificate} />)}
          </ul>
        ) : <p>Справок пока нет.</p>}
      </ReadState>
    </section>
  );
}

function CertificateRow({ certificate }: { certificate: Certificate }) {
  return (
    <li className={`${panel} space-y-2`}>
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <Link href={orderPath(certificate.order_id)} className={link}>Заказ {certificate.order_id.slice(0, 8)}</Link>
        <time dateTime={certificate.created_at} className="text-sm text-muted-foreground">
          {dateFormat.format(new Date(certificate.created_at))}
        </time>
      </div>
      <p>{certificateStatusLabels[certificate.status]}</p>
      {certificate.status === "ready" && <CertificateDownload id={certificate.order_id} />}
    </li>
  );
}
