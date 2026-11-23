import { VehicleScreen } from "@/features/customer/vehicles/view";

export default function VehiclePage({ params }: { params: Promise<{ id: string }> }) {
  return params.then(({ id }) => <VehicleScreen id={id} />);
}
