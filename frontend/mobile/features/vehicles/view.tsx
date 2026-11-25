import { useCallback, useState } from "react";
import { Button, FlatList, Text, View } from "react-native";
import { EngineerApiError } from "../../common/api/controller";
import { useLoad } from "../../common/hooks/use-load";
import { HistoryRow } from "../../common/order-history/view";
import { styles } from "../../common/styles";
import { InspectionCamera } from "./camera/view";
import { loadVehicle, loadVehicleHistory } from "./controller";
import type { VehiclePhoto } from "./photos/controller";
import { PhotoPreview } from "./photos/view";

export function VehicleScreen({ token, id, orderId, historyOrderId, onBack, onHistoryOrderOpened, onSessionEnd }: {
  token: string;
  id: string;
  orderId?: string;
  historyOrderId?: string;
  onBack: () => void;
  onHistoryOrderOpened: (id: string) => void;
  onSessionEnd: (message: string) => void;
}) {
  const load = useCallback((signal: AbortSignal) => loadVehicle(token, id, signal), [id, token]);
  const { state, reload } = useLoad(load, onSessionEnd);
  const [selected, setSelected] = useState<{ photo: VehiclePhoto; label: string }>();
  const [cameraOpen, setCameraOpen] = useState(false);

  return (
    <>
      <Button title="Назад" onPress={onBack} />
      {state.status === "loading" && <Text accessibilityLiveRegion="polite">Загружаем карточку ТС…</Text>}
      {state.status === "error" && (
        <>
          <Text accessibilityRole="alert" style={styles.error}>
            {state.error instanceof EngineerApiError && state.error.status === 404
              ? "Автомобиль не найден."
              : "Не удалось загрузить карточку ТС."}
          </Text>
          <Button title="Повторить загрузку" onPress={reload} />
        </>
      )}
      {state.status === "ready" && (
        <FlatList data={state.data.photos} keyExtractor={(photo) => photo.photo_id}
          ListHeaderComponent={(
            <View style={{ gap: 10, paddingBottom: 12 }}>
              <Text accessibilityRole="header" style={styles.heading}>{state.data.brand} · {state.data.number}</Text>
              <Text>VIN {state.data.vin}</Text>
              {state.data.cover_photo_url && <Button title="Посмотреть обложку"
                onPress={() => setSelected({ photo: { kind: "cover" }, label: "Обложка автомобиля" })} />}
              {orderId && <Button title="Сделать снимок" onPress={() => setCameraOpen(true)} />}
              <Text accessibilityRole="header">Фотофиксация</Text>
            </View>
          )}
          renderItem={({ item, index }) => (
            <View style={{ paddingBottom: 8 }}>
              <Button title={`Открыть снимок ${index + 1}`} onPress={() => setSelected({
                photo: { kind: "gallery", id: item.photo_id }, label: `Снимок ${index + 1}`,
              })} />
            </View>
          )}
          ListEmptyComponent={<Text>Снимков фотофиксации пока нет.</Text>}
          ListFooterComponent={historyOrderId ? <VehicleHistory token={token} orderId={historyOrderId}
            onOpen={onHistoryOrderOpened} onSessionEnd={onSessionEnd} /> : undefined} />
      )}
      {selected && <PhotoPreview token={token} vehicleId={id} photo={selected.photo} label={selected.label}
        onClose={() => setSelected(undefined)} onSessionEnd={onSessionEnd} />}
      {cameraOpen && orderId && <InspectionCamera token={token} vehicleId={id} orderId={orderId}
        onClose={() => setCameraOpen(false)} onSessionEnd={onSessionEnd} onSaved={() => {
          setCameraOpen(false);
          reload();
        }} />}
    </>
  );
}

function VehicleHistory({ token, orderId, onOpen, onSessionEnd }: {
  token: string;
  orderId: string;
  onOpen: (id: string) => void;
  onSessionEnd: (message: string) => void;
}) {
  const load = useCallback((signal: AbortSignal) => loadVehicleHistory(token, orderId, signal), [token, orderId]);
  const { state, reload } = useLoad(load, onSessionEnd);

  return (
    <View style={{ gap: 8, paddingTop: 16 }}>
      <Text accessibilityRole="header">История работ</Text>
      {state.status === "loading" && <Text accessibilityLiveRegion="polite">Загружаем историю…</Text>}
      {state.status === "error" && (
        <>
          <Text accessibilityRole="alert" style={styles.error}>
            {state.error instanceof EngineerApiError && state.error.status === 404
              ? "История доступна ответственному по текущему заказу."
              : "Не удалось загрузить историю работ."}
          </Text>
          <Button title="Обновить историю" onPress={reload} />
        </>
      )}
      {state.status === "ready" && (state.data.length
        ? state.data.map((order) => <HistoryRow key={order.id} order={order} onOpen={onOpen} />)
        : <Text>Других заказов по этому ТС пока нет.</Text>)}
    </View>
  );
}
