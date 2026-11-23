import { CameraView, useCameraPermissions } from "expo-camera";
import { useEffect, useRef, useState } from "react";
import { AppState, Button, Image, Linking, Modal, StyleSheet, Text, View } from "react-native";
import { EngineerApiError, sessionErrorMessage } from "../../../common/api/controller";
import { styles } from "../../../common/styles";
import { uploadVehiclePhoto } from "./controller";
import { discardInspectionPhoto, prepareInspectionPhoto } from "./form";

export function InspectionCamera({ token, vehicleId, orderId, onClose, onSaved, onSessionEnd }: {
  token: string;
  vehicleId: string;
  orderId: string;
  onClose: () => void;
  onSaved: () => void;
  onSessionEnd: (message: string) => void;
}) {
  const [permission, requestPermission, getPermission] = useCameraPermissions();
  const [ready, setReady] = useState(false);
  const [captureAttempt, setCaptureAttempt] = useState(0);
  const [photoUri, setPhotoUri] = useState<string>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const camera = useRef<CameraView>(null);
  const operation = useRef<AbortController | null>(null);

  useEffect(() => () => operation.current?.abort(), []);
  useEffect(() => () => {
    if (photoUri) discardInspectionPhoto(photoUri);
  }, [photoUri]);

  useEffect(() => {
    if (permission?.canAskAgain !== false || permission.granted) return;

    const subscription = AppState.addEventListener("change", (state) => {
      if (state === "active") void getPermission().catch(() => undefined);
    });
    return () => subscription.remove();
  }, [getPermission, permission]);

  function run(action: (signal: AbortSignal) => Promise<void>, describeError: (error: unknown) => string) {
    if (operation.current) return;
    const controller = new AbortController();
    operation.current = controller;
    setBusy(true);
    setError("");
    void action(controller.signal).catch((cause: unknown) => {
      if (controller.signal.aborted) return;
      const sessionError = sessionErrorMessage(cause);
      if (sessionError) return onSessionEnd(sessionError);
      setError(describeError(cause));
    }).finally(() => {
      operation.current = null;
      if (!controller.signal.aborted) setBusy(false);
    });
  }

  function takePhoto() {
    const currentCamera = camera.current;
    if (!currentCamera || !ready) return;

    run(async (signal) => {
      const capture = await currentCamera.takePictureAsync({ quality: 0.85 });
      try {
        if (signal.aborted) return;
        const uri = await prepareInspectionPhoto(capture);
        if (signal.aborted) discardInspectionPhoto(uri);
        else setPhotoUri(uri);
      } finally {
        discardInspectionPhoto(capture.uri);
      }
    }, () => "Не удалось сделать снимок. Попробуйте ещё раз.");
  }

  function retake() {
    setError("");
    setReady(false);
    setPhotoUri(undefined);
    setCaptureAttempt((attempt) => attempt + 1);
  }

  function save() {
    if (!photoUri) return;
    run(async (signal) => {
      await uploadVehiclePhoto(token, vehicleId, orderId, photoUri, signal);
      if (!signal.aborted) onSaved();
    }, (cause) => cause instanceof EngineerApiError && cause.status === 413
      ? "Снимок слишком большой. Переснимите его."
      : "Не удалось отправить снимок. Попробуйте ещё раз.");
  }

  function close() {
    if (!operation.current) onClose();
  }

  function allowCamera() {
    run(async () => {
      if (permission?.canAskAgain === false) {
        await Linking.openSettings();
      } else {
        await requestPermission();
      }
    }, () => "Не удалось получить доступ к камере.");
  }

  return (
    <Modal visible onRequestClose={close}>
      <View style={styles.screen}>
        <Button title="Закрыть камеру" onPress={close} disabled={busy} />
        <Text accessibilityRole="header" style={styles.heading}>Фотофиксация</Text>
        {photoUri ? (
          <>
            <Image accessibilityLabel="Предпросмотр снимка" source={{ uri: photoUri }}
              style={cameraStyles.preview} resizeMode="contain" />
            <Button title="Переснять" onPress={retake} disabled={busy} />
            <Button title={busy ? "Отправляем…" : "Отправить снимок"} onPress={save} disabled={busy} />
          </>
        ) : permission?.granted ? (
          <>
            <CameraView key={captureAttempt} ref={camera} style={cameraStyles.preview} facing="back"
              onCameraReady={() => setReady(true)} onMountError={() => {
                setReady(false);
                setError("Камера недоступна. Попробуйте открыть её снова.");
              }} />
            <Button title={busy ? "Готовим снимок…" : "Сделать снимок"} onPress={takePhoto}
              disabled={!ready || busy} />
            {!ready && error && <Button title="Перезапустить камеру" onPress={retake} disabled={busy} />}
          </>
        ) : permission ? (
          <>
            <Text>Разрешите доступ к камере для фотофиксации.</Text>
            <Button title={permission.canAskAgain ? "Разрешить доступ" : "Открыть настройки"}
              onPress={allowCamera} disabled={busy} />
          </>
        ) : <Text accessibilityLiveRegion="polite">Проверяем доступ к камере…</Text>}
        {error ? <Text accessibilityRole="alert" style={styles.error}>{error}</Text> : null}
      </View>
    </Modal>
  );
}

const cameraStyles = StyleSheet.create({
  preview: { flex: 1, width: "100%" },
});
