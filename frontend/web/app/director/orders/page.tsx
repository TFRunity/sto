import { Suspense } from "react";
import { DirectorOrderSearch } from "@/features/director/orders/search/view";

export default function DirectorOrdersPage() {
  return (
    <Suspense fallback={<p role="status">Загружаем заказы…</p>}>
      <DirectorOrderSearch />
    </Suspense>
  );
}
