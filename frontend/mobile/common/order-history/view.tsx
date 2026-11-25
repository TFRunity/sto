import { Button, Text, View } from "react-native";
import type { components } from "../../../common/api/schema";
import { orderStageLabels, orderStatusLabels } from "../../../common/orders/labels";

export type HistoryOrder = components["schemas"]["OrderHistoryItem"];

export function HistoryRow({ order, onOpen }: { order: HistoryOrder; onOpen: (id: string) => void }) {
  const date = new Date(order.created_at).toLocaleDateString("ru-RU");
  const vehicle = [order.vehicle_brand, order.vehicle_number].filter(Boolean).join(" · ");

  return (
    <View style={{ gap: 4, paddingVertical: 8 }}>
      <Button title={`Заказ ${order.id.slice(0, 8)}`} onPress={() => onOpen(order.id)} />
      {vehicle ? <Text>{vehicle}</Text> : null}
      <Text>{date} · {orderStatusLabels[order.status]} · {orderStageLabels[order.stage]}</Text>
    </View>
  );
}
