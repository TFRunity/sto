import { BookingScreen } from "@/features/customer/booking/view";

export default function VehicleBookingPage({ params }: { params: Promise<{ id: string }> }) {
  return params.then(({ id }) => <BookingScreen id={id} />);
}
