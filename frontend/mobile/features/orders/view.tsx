import { useCallback, useEffect, useRef, useState } from "react";
import { Alert, Button, ScrollView, Text, TextInput, View } from "react-native";
import { formatRubles } from "../../../common/money/utils";
import { orderStageLabels, orderStatusLabels } from "../../../common/orders/labels";
import { certificateStatusLabels } from "../../../common/orders/certificate";
import { isPaymentPending } from "../../../common/orders/payment";
import { EngineerApiError, sessionErrorMessage } from "../../common/api/controller";
import { useLoad } from "../../common/hooks/use-load";
import { styles } from "../../common/styles";
import { loadOrder, updateOrder, type Order, type OrderWrite } from "./controller";
import { availableStages, canCancelOrder, readPositionForm } from "./form";
import type { PositionKind } from "./search/form";
import { PositionSearch } from "./search/view";

export function OrderScreen({ token, id, historyAnchorId, onBack, onSessionEnd, onVehicleOpened }: {
  token: string;
  id: string;
  historyAnchorId?: string;
  onBack: () => void;
  onVehicleOpened: (vehicleId: string, withHistory: boolean, canEdit: boolean) => void;
  onSessionEnd: (message?: string) => void;
}) {
  const load = useCallback((signal: AbortSignal) => loadOrder(token, id, signal, historyAnchorId), [historyAnchorId, id, token]);
  const { state, reload } = useLoad(load, onSessionEnd);
  const order = state.status === "ready" ? state.data : undefined;
  const canEdit = !historyAnchorId && order?.can_edit === true;
  const confirming = order && isPaymentPending(order);
  const [notice, setNotice] = useState("");
  const [saving, setSaving] = useState(false);
  const writeRequest = useRef<AbortController | null>(null);

  useEffect(() => () => writeRequest.current?.abort(), []);

  function save(write: OrderWrite) {
    if (writeRequest.current || !canEdit) return;
    const controller = new AbortController();
    writeRequest.current = controller;
    setSaving(true);
    setNotice("");

    void updateOrder(token, id, write, controller.signal).then(() => {
      if (controller.signal.aborted) return;
      reload();
    }).catch((cause: unknown) => {
      if (controller.signal.aborted) return;
      const sessionError = sessionErrorMessage(cause);
      if (sessionError) return onSessionEnd(sessionError);

      if (cause instanceof EngineerApiError && cause.status === 409) {
        setNotice("Изменение отклонено из-за состояния заказа. Перечитываем данные.");
        reload();
        return;
      }
      if (!(cause instanceof EngineerApiError) || cause.status >= 500) {
        setNotice("Не удалось подтвердить изменение. Перечитываем заказ перед повтором.");
        reload();
        return;
      }
      setNotice("Не удалось изменить заказ. Проверьте данные и попробуйте ещё раз.");
    }).finally(() => {
      writeRequest.current = null;
      if (!controller.signal.aborted) setSaving(false);
    });
  }

  return (
    <ScrollView keyboardShouldPersistTaps="handled" contentContainerStyle={{ gap: 12 }}>
      <Button title="Назад" onPress={onBack} />
      <Text accessibilityRole="header" style={styles.heading}>Заказ {id.slice(0, 8)}</Text>
      {notice ? <Text accessibilityRole="alert" style={styles.error}>{notice}</Text> : null}
      {state.status === "error" ? (
        <>
          <Text accessibilityRole="alert" style={styles.error}>
            {state.error instanceof EngineerApiError && state.error.status === 404
              ? "Заказ больше недоступен."
              : "Не удалось загрузить заказ."}
          </Text>
          <Button title="Повторить загрузку" onPress={reload} />
        </>
      ) : order ? (
        <>
          <Text accessibilityLiveRegion="polite">
            Статус: {confirming ? "Оплата подтверждается" : orderStatusLabels[order.status]}
          </Text>
          <Text>Этап: {orderStageLabels[order.stage]}</Text>
          {order.certificate_status && <Text>Справка: {certificateStatusLabels[order.certificate_status]}</Text>}
          {order.stock_status === "pending" && <Text accessibilityLiveRegion="polite">Ожидаем списание товара со склада.</Text>}
          {order.stock_status === "rejected" && (
            <>
              <Text accessibilityRole="alert">На складе не хватило товара. Проверьте остатки перед повтором.</Text>
              {canEdit && <Button title="Повторить списание" onPress={() => save({ action: "stock/retry" })} disabled={saving} />}
            </>
          )}
          <Button title="Карточка ТС" disabled={saving} onPress={() => onVehicleOpened(order.vehicle_id,
            canEdit && (order.status === "active" || (order.status === "paid" && order.stage !== "выдан")), canEdit)} />
          {order.engineers && order.engineers.length > 0 && (
            <View style={{ gap: 4 }}>
              <Text accessibilityRole="header">Исполнители</Text>
              {order.engineers.map((engineer) => (
                <Text key={engineer.id}>
                  {engineer.name || engineer.id}{engineer.id === order.engineer_id ? " · Ответственный" : ""}
                </Text>
              ))}
            </View>
          )}
          <OrderPositions order={order} />
          <Text>Итого: {formatRubles(order.total)}</Text>
          {canEdit && order.status === "active" && (
            <PositionEditor token={token} saving={saving} onSave={save} onSessionEnd={onSessionEnd} />
          )}
          {canEdit && availableStages(order).filter((stage) => stage !== order.stage).map((stage) => (
            <Button key={stage} title={stage === "выдан" ? "Выдать автомобиль" : `Этап: ${orderStageLabels[stage]}`} disabled={saving}
              onPress={() => save({ action: "stage", body: { stage } })} />
          ))}
          {canEdit && canCancelOrder(order) && (
            <Button title="Отменить заказ" disabled={saving} onPress={() => {
              Alert.alert("Отменить заказ?", "Заказ перейдёт в статус «Отменён».", [
                { text: "Назад", style: "cancel" },
                { text: "Отменить заказ", style: "destructive", onPress: () => save({ action: "cancel" }) },
              ]);
            }} />
          )}
          {saving && <Text accessibilityLiveRegion="polite">Сохраняем изменения…</Text>}
          <Button title="Обновить заказ" onPress={reload} disabled={saving} />
        </>
      ) : <Text accessibilityLiveRegion="polite">Загружаем заказ…</Text>}
    </ScrollView>
  );
}

function OrderPositions({ order }: { order: Order }) {
  return (
    <View style={{ gap: 6 }}>
      <Text accessibilityRole="header">Позиции заказа</Text>
      {order.items.length === 0 && order.services.length === 0 && <Text>Позиций пока нет.</Text>}
      {order.items.map((item) => (
        <Text key={item.id}>{item.name || `Товар ${item.item_id.slice(0, 8)}`} × {item.quantity} · {formatRubles(item.price)} за шт.</Text>
      ))}
      {order.services.map((service) => (
        <Text key={service.id}>{service.name} × {service.quantity} · {formatRubles(service.price)} за ед.</Text>
      ))}
    </View>
  );
}

function PositionEditor({ token, saving, onSave, onSessionEnd }: {
  token: string;
  saving: boolean;
  onSave: (write: OrderWrite) => void;
  onSessionEnd: (message?: string) => void;
}) {
  const [kind, setKind] = useState<PositionKind>();
  const [selected, setSelected] = useState<{ id: string; name: string }>();
  const [quantity, setQuantity] = useState("1");
  const [error, setError] = useState("");

  function open(next: PositionKind) {
    setKind(next);
    setSelected(undefined);
    setQuantity("1");
    setError("");
  }

  function submit() {
    if (!selected || !kind) return;
    const write = readPositionForm(kind, selected.id, quantity);
    if (!write) {
      setError("Укажите целое количество от 1 до 2 147 483 647.");
      return;
    }

    onSave(write);
  }

  return (
    <View style={{ gap: 8 }}>
      <Button title="Добавить товар" onPress={() => open("item")} disabled={saving} />
      <Button title="Добавить услугу" onPress={() => open("service")} disabled={saving} />
      {kind && (selected ? (
        <>
          <Text>Выбрано: {selected.name}</Text>
          <Text>Количество</Text>
          <TextInput style={styles.input} accessibilityLabel="Количество" keyboardType="number-pad"
            value={quantity} editable={!saving} onChangeText={(value) => {
              setQuantity(value);
              setError("");
            }} />
          {error ? <Text accessibilityRole="alert" style={styles.error}>{error}</Text> : null}
          <Button title="Добавить позицию" onPress={submit} disabled={saving} />
          <Button title="Выбрать другую позицию" onPress={() => open(kind)} disabled={saving} />
        </>
      ) : (
        <PositionSearch key={kind} token={token} kind={kind} onSelect={setSelected}
          onSessionEnd={onSessionEnd} />
      ))}
    </View>
  );
}
