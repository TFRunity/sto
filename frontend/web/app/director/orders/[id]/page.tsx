import { DirectorOrderScreen } from "@/features/director/orders/details/view";

export default async function DirectorOrderPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <DirectorOrderScreen id={id} />;
}
