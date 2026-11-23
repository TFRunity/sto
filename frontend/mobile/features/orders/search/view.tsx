import { useCallback, useState } from "react";
import { Button, Text, TextInput, View } from "react-native";
import { formatRubles } from "../../../../common/money/utils";
import { useLoad } from "../../../common/hooks/use-load";
import { styles } from "../../../common/styles";
import { filterPositions, type PositionKind } from "./form";
import { loadInventory } from "./inventory/controller";
import { loadServices } from "./services/controller";

type CatalogEntry = { id: string; name: string; price: string; quantity?: number };

export function PositionSearch({ token, kind, onSelect, onSessionEnd }: {
  token: string;
  kind: PositionKind;
  onSelect: (entry: CatalogEntry) => void;
  onSessionEnd: (message?: string) => void;
}) {
  const [input, setInput] = useState("");
  const load = useCallback((signal: AbortSignal) => (
    kind === "item" ? loadInventory(token, signal) : loadServices(token, signal)
  ), [kind, token]);
  const { state, reload } = useLoad<CatalogEntry[]>(load, onSessionEnd);

  if (state.status === "error") return (
    <View style={{ gap: 8 }}>
      <Text accessibilityRole="alert" style={styles.error}>Не удалось загрузить каталог.</Text>
      <Button title="Повторить загрузку каталога" onPress={reload} />
    </View>
  );

  if (state.status === "loading") return <Text accessibilityLiveRegion="polite">Загружаем каталог…</Text>;

  const matches = filterPositions(state.data, input);
  const searchLabel = kind === "item" ? "Поиск товара" : "Поиск услуги";

  return (
    <View style={{ gap: 8 }}>
      <TextInput style={styles.input} accessibilityLabel={searchLabel}
        placeholder={searchLabel} value={input} onChangeText={setInput} />
      {matches.length === 0 && <Text>Подходящих позиций нет.</Text>}
      {matches.slice(0, 20).map((entry) => (
        <View key={entry.id} style={{ gap: 4 }}>
          <Text>{entry.name} · {formatRubles(entry.price)}</Text>
          {entry.quantity !== undefined && <Text>В наличии: {entry.quantity}</Text>}
          <Button title={`Выбрать: ${entry.name}`} onPress={() => onSelect(entry)} />
        </View>
      ))}
      {matches.length > 20 && <Text>Уточните поиск, чтобы увидеть остальные позиции.</Text>}
    </View>
  );
}
