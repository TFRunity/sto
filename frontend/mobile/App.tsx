import { useCallback, useEffect, useState } from "react";
import { BackHandler, StatusBar, View } from "react-native";
import { styles } from "./common/styles";
import { EngineerLogin } from "./features/auth/view";
import { OrderScreen } from "./features/orders/view";
import { BookingPool } from "./features/pool/view";
import { VehicleScreen } from "./features/vehicles/view";

export default function App() {
  const [token, setToken] = useState<string | null>(null);
  const [orderId, setOrderId] = useState<string | null>(null);
  const [vehicle, setVehicle] = useState<{ id: string; orderId?: string } | null>(null);
  const [sessionMessage, setSessionMessage] = useState("");
  const endSession = useCallback((message = "") => {
    setVehicle(null);
    setOrderId(null);
    setToken(null);
    setSessionMessage(message);
  }, []);

  const closeOrder = useCallback(() => setOrderId(null), []);

  const closeVehicle = useCallback(() => setVehicle(null), []);

  useEffect(() => {
    if (!vehicle && !orderId) return;
    const subscription = BackHandler.addEventListener("hardwareBackPress", () => {
      if (vehicle) closeVehicle();
      else closeOrder();
      return true;
    });

    return () => subscription.remove();
  }, [closeOrder, closeVehicle, orderId, vehicle]);

  function renderScreen() {
    if (!token) return <EngineerLogin sessionMessage={sessionMessage} onSignIn={setToken} />;
    if (vehicle) {
      return <VehicleScreen token={token} id={vehicle.id} orderId={vehicle.orderId}
        onBack={closeVehicle} onSessionEnd={endSession} />;
    }
    if (orderId) {
      return <OrderScreen token={token} id={orderId} onBack={closeOrder} onSessionEnd={endSession}
        onVehicleOpened={(id) => setVehicle({ id, orderId })} />;
    }
    return <BookingPool token={token} onOrderOpened={setOrderId} onSessionEnd={endSession}
      onVehicleOpened={(id) => setVehicle({ id })} />;
  }

  return (
    <View style={styles.screen}>
      <StatusBar barStyle="dark-content" />
      {renderScreen()}
    </View>
  );
}
