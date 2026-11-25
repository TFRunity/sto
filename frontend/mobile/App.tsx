import { useCallback, useEffect, useState } from "react";
import { BackHandler, StatusBar, View } from "react-native";
import { styles } from "./common/styles";
import { AssignedOrders } from "./features/assigned-orders/view";
import { EngineerLogin } from "./features/auth/view";
import { IntakeScreen } from "./features/intake/view";
import { OrderScreen } from "./features/orders/view";
import { BookingPool } from "./features/pool/view";
import { EngineerProfile } from "./features/profile/view";
import { VehicleScreen } from "./features/vehicles/view";

type Screen =
  | { kind: "assigned" | "intake" | "profile" }
  | { kind: "order"; id: string; historyAnchorId?: string }
  | { kind: "vehicle"; id: string; orderId?: string; historyOrderId?: string };

export default function App() {
  const [session, setSession] = useState<{ token: string; phone: string } | null>(null);
  const [screens, setScreens] = useState<Screen[]>([]);
  const screen = screens.at(-1);
  const [sessionMessage, setSessionMessage] = useState("");
  const endSession = useCallback((message = "") => {
    setScreens([]);
    setSession(null);
    setSessionMessage(message);
  }, []);

  const goBack = useCallback(() => setScreens((history) => history.slice(0, -1)), []);

  useEffect(() => {
    if (!screen) return;
    const subscription = BackHandler.addEventListener("hardwareBackPress", () => {
      goBack();
      return true;
    });

    return () => subscription.remove();
  }, [goBack, screen]);

  function open(next: Screen) {
    setScreens((history) => [...history, next]);
  }

  function openOrder(id: string) {
    open({ kind: "order", id });
  }

  function openAcceptedOrder(id: string) {
    setScreens([{ kind: "assigned" }, { kind: "order", id }]);
  }

  function renderScreen() {
    if (!session) return <EngineerLogin sessionMessage={sessionMessage}
      onSignIn={(token, phone) => setSession({ token, phone })} />;
    const { token } = session;
    switch (screen?.kind) {
      case "vehicle":
        return <VehicleScreen key={screen.id} token={token} id={screen.id} orderId={screen.orderId}
          historyOrderId={screen.historyOrderId} onBack={goBack} onSessionEnd={endSession}
          onHistoryOrderOpened={(id) => open({ kind: "order", id, historyAnchorId: screen.historyOrderId })} />;
      case "order":
        return <OrderScreen key={screen.id} token={token} id={screen.id} historyAnchorId={screen.historyAnchorId}
          onBack={goBack} onSessionEnd={endSession} onVehicleOpened={(id, withHistory, canEdit) => open({
            kind: "vehicle", id,
            orderId: canEdit ? screen.id : undefined,
            historyOrderId: withHistory ? screen.id : undefined,
          })} />;
      case "assigned":
        return <AssignedOrders token={token} onBack={goBack} onOrderOpened={openOrder} onSessionEnd={endSession} />;
      case "intake":
        return <IntakeScreen token={token} engineerPhone={session.phone} onBack={goBack}
          onOrderOpened={openAcceptedOrder} onSessionEnd={endSession} />;
      case "profile":
        return <EngineerProfile token={token} onBack={goBack} onSessionEnd={endSession} />;
      default:
        return <BookingPool token={token} onOrderOpened={openAcceptedOrder} onSessionEnd={endSession}
          onVehicleOpened={(id) => open({ kind: "vehicle", id })}
          onAssignedOpened={() => open({ kind: "assigned" })} onIntakeOpened={() => open({ kind: "intake" })}
          onProfileOpened={() => open({ kind: "profile" })} />;
    }
  }

  return (
    <View style={styles.screen}>
      <StatusBar barStyle="dark-content" />
      {renderScreen()}
    </View>
  );
}
