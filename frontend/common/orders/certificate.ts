import type { components } from "../api/schema";

export const certificateStatusLabels = {
  processing: "Формируется",
  ready: "Готова",
  failed: "Не удалось сформировать",
} satisfies Record<components["schemas"]["CertificateStatus"], string>;
