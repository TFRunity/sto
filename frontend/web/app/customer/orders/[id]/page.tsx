import { OrderScreen } from "@/features/customer/orders/view";

export default function OrderPage({ params }: { params: Promise<{ id: string }> }) {
  return params.then(({ id }) => <OrderScreen id={id} />);
}
