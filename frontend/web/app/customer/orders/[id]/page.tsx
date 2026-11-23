import { OrderScreen } from "@/features/customer/customer-screens";

export default function OrderPage({ params }: { params: Promise<{ id: string }> }) {
  return params.then(({ id }) => <OrderScreen id={id} />);
}
