import { useCallback, useState } from "react";
import { Button, FlatList, Text, View } from "react-native";
import { useLoad } from "../../common/hooks/use-load";
import { HistoryRow } from "../../common/order-history/view";
import { styles } from "../../common/styles";
import { loadAssignedOrders, type OrderScope } from "./controller";

export function AssignedOrders({ token, onBack, onOrderOpened, onSessionEnd }: {
  token: string;
  onBack: () => void;
  onOrderOpened: (id: string) => void;
  onSessionEnd: (message: string) => void;
}) {
  const [scope, setScope] = useState<OrderScope>("active");
  const load = useCallback((signal: AbortSignal) => loadAssignedOrders(token, scope, signal), [token, scope]);
  const { state, reload } = useLoad(load, onSessionEnd);

  function changeScope(nextScope: OrderScope) {
    reload();
    setScope(nextScope);
  }

  return (
    <>
      <Button title="К пулу заявок" onPress={onBack} />
      <Text accessibilityRole="header" style={styles.heading}>Мои заказы</Text>
      <View style={styles.header}>
        <Button title="Активные" disabled={scope === "active"} onPress={() => changeScope("active")} />
        <Button title="Завершённые" disabled={scope === "completed"} onPress={() => changeScope("completed")} />
      </View>
      {state.status === "loading" && <Text accessibilityLiveRegion="polite">Загружаем заказы…</Text>}
      {state.status === "error" && (
        <>
          <Text accessibilityRole="alert" style={styles.error}>Не удалось загрузить заказы.</Text>
          <Button title="Повторить загрузку" onPress={reload} />
        </>
      )}
      {state.status === "ready" && (
        <FlatList data={state.data} keyExtractor={(item) => item.id}
          renderItem={({ item }) => <HistoryRow order={item} onOpen={onOrderOpened} />}
          ListEmptyComponent={<Text>Заказов пока нет.</Text>}
          onRefresh={reload} refreshing={false} />
      )}
    </>
  );
}
