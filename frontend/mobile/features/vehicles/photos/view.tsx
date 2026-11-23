import { useCallback, useState } from "react";
import { Button, Image, Modal, StyleSheet, Text, View } from "react-native";
import { useLoad } from "../../../common/hooks/use-load";
import { styles } from "../../../common/styles";
import { loadVehiclePhoto, type VehiclePhoto } from "./controller";

export function PhotoPreview({ token, vehicleId, photo, label, onClose, onSessionEnd }: {
  token: string;
  vehicleId: string;
  photo: VehiclePhoto;
  label: string;
  onClose: () => void;
  onSessionEnd: (message: string) => void;
}) {
  const load = useCallback((signal: AbortSignal) => (
    loadVehiclePhoto(token, vehicleId, photo, signal)
  ), [photo, token, vehicleId]);
  const { state, reload } = useLoad(load, onSessionEnd);
  const [decodeFailed, setDecodeFailed] = useState(false);

  return (
    <Modal visible onRequestClose={onClose}>
      <View style={styles.screen}>
        <Button title="Закрыть" onPress={onClose} />
        <Text accessibilityRole="header" style={styles.heading}>{label}</Text>
        {state.status === "error" || decodeFailed ? (
          <>
            <Text accessibilityRole="alert" style={styles.error}>Не удалось открыть фотографию.</Text>
            <Button title="Повторить загрузку" onPress={() => {
              setDecodeFailed(false);
              reload();
            }} />
          </>
        ) : state.status === "ready" ? (
          <Image accessibilityLabel={label} source={{ uri: state.data }}
            style={photoStyles.image} resizeMode="contain" onError={() => setDecodeFailed(true)} />
        ) : <Text accessibilityLiveRegion="polite">Загружаем фотографию…</Text>}
      </View>
    </Modal>
  );
}

const photoStyles = StyleSheet.create({
  image: { flex: 1, width: "100%" },
});
