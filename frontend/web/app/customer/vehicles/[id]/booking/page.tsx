import { VehicleScreen } from "@/features/customer/customer-screens";

export default function VehicleBookingPage({ params }: { params: Promise<{ id: string }> }) {
  return params.then(({ id }) => <VehicleScreen id={id} booking />);
}
