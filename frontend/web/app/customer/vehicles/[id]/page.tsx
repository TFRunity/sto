import { VehicleScreen } from "@/features/customer/screens";

export default function VehiclePage({ params }: { params: Promise<{ id: string }> }) {
  return params.then(({ id }) => <VehicleScreen id={id} />);
}
