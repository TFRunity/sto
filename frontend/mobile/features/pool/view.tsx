import { useCallback, useEffect, useRef, useState } from "react";
import { Button, FlatList, StyleSheet, Text, View } from "react-native";
import { EngineerApiError, sessionErrorMessage } from "../../common/api/controller";
import { useLoad } from "../../common/hooks/use-load";
import { styles } from "../../common/styles";
import { claimBooking, loadBookingPool } from "./controller";

export function BookingPool({ token, onOrderOpened, onSessionEnd, onVehicleOpened, onAssignedOpened, onIntakeOpened, onProfileOpened }: {
  token: string;
  onOrderOpened: (id: string) => void;
  onVehicleOpened: (vehicleId: string) => void;
  onSessionEnd: (message?: string) => void;
  onAssignedOpened: () => void;
  onIntakeOpened: () => void;
  onProfileOpened: () => void;
}) {
  const load = useCallback((signal: AbortSignal) => loadBookingPool(token, signal), [token]);
  const { state, reload } = useLoad(load, onSessionEnd);
  const [claimingId, setClaimingId] = useState<string>();
  const [message, setMessage] = useState("");
  const claim = useRef<AbortController | null>(null);

  useEffect(() => () => claim.current?.abort(), []);

  function takeBooking(id: string) {
    if (claim.current) return;
    const controller = new AbortController();
    claim.current = controller;
    setClaimingId(id);
    setMessage("");

    void claimBooking(token, id, controller.signal).then(() => {
      if (controller.signal.aborted) return;
      onOrderOpened(id);
    }).catch((error: unknown) => {
      if (controller.signal.aborted) return;
      const sessionError = sessionErrorMessage(error);
      if (sessionError) return onSessionEnd(sessionError);

      if (error instanceof EngineerApiError && [404, 409].includes(error.status)) {
        setMessage("Заявка больше недоступна. Обновляем пул.");
        reload();
        return;
      }
      setMessage("Не удалось взять заявку. Попробуйте ещё раз.");
    }).finally(() => {
      claim.current = null;
      if (!controller.signal.aborted) setClaimingId(undefined);
    });
  }

  return (
    <>
      <View style={styles.header}>
        <Text accessibilityRole="header" style={styles.heading}>Пул заявок</Text>
        <Button title="Выйти" onPress={() => onSessionEnd()} />
      </View>
      <Button title="Мои заказы" onPress={onAssignedOpened} disabled={Boolean(claimingId)} />
      <Button title="Прямой приём" onPress={onIntakeOpened} disabled={Boolean(claimingId)} />
      <Button title="Профиль" onPress={onProfileOpened} disabled={Boolean(claimingId)} />
      {message ? <Text accessibilityRole="alert" style={styles.error}>{message}</Text> : null}
      {state.status === "loading" && <Text accessibilityLiveRegion="polite">Загружаем заявки…</Text>}
      {state.status === "error" && (
        <>
          <Text accessibilityRole="alert" style={styles.error}>Не удалось загрузить пул заявок.</Text>
          <Button title="Повторить загрузку" onPress={reload} />
        </>
      )}
      {state.status === "ready" && (
        <FlatList
          data={state.data}
          keyExtractor={(order) => order.id}
          renderItem={({ item }) => (
            <View style={poolStyles.row}>
              <Text accessibilityRole="header">{item.service || `Заявка ${item.id.slice(0, 8)}`}</Text>
              {item.date ? <Text>Желаемая дата: {item.date.split("-").reverse().join(".")}</Text> : null}
              {item.comment ? <Text>{item.comment}</Text> : null}
              <Button title="Карточка ТС" onPress={() => onVehicleOpened(item.vehicle_id)} disabled={Boolean(claimingId)} />
              <Button title={claimingId === item.id ? "Берём в работу…" : "Взять в работу"}
                disabled={Boolean(claimingId)} onPress={() => takeBooking(item.id)} />
            </View>
          )}
          ListEmptyComponent={<Text>Открытых заявок пока нет.</Text>}
        />
      )}
    </>
  );
}

const poolStyles = StyleSheet.create({
  row: { gap: 8, paddingVertical: 12 },
});
