import { useCallback, useEffect, useRef, useState } from "react";
import { Button, ScrollView, Switch, Text } from "react-native";
import { formatProfileDate, profileRoleLabels } from "../../../common/profile/labels";
import { sessionErrorMessage } from "../../common/api/controller";
import { useLoad } from "../../common/hooks/use-load";
import { styles } from "../../common/styles";
import { loadProfile, updateNotifications } from "./controller";

export function EngineerProfile({ token, onBack, onSessionEnd, onNotificationsChanged }: {
  token: string;
  onBack: () => void;
  onSessionEnd: (message?: string) => void;
  onNotificationsChanged: () => void;
}) {
  const load = useCallback((signal: AbortSignal) => loadProfile(token, signal), [token]);
  const { state, reload } = useLoad(load, onSessionEnd);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const request = useRef<AbortController | null>(null);

  useEffect(() => () => request.current?.abort(), []);

  async function save(enabled: boolean) {
    if (request.current) return;
    const controller = new AbortController();
    request.current = controller;
    setSaving(true);
    setMessage("");
    try {
      await updateNotifications(token, enabled, controller.signal);
    } catch (cause: unknown) {
      if (controller.signal.aborted) return;
      const sessionError = sessionErrorMessage(cause);
      if (sessionError) {
        onSessionEnd(sessionError);
        return;
      }
      setMessage("Не удалось подтвердить сохранение настройки.");
    } finally {
      request.current = null;
      if (!controller.signal.aborted) {
        // Даже после потери ответа сервер мог сохранить настройку. Перечитываем профиль.
        reload();
        onNotificationsChanged();
        setSaving(false);
      }
    }
  }

  return (
    <ScrollView contentContainerStyle={{ gap: 12 }}>
      <Button title="Назад" onPress={onBack} />
      <Text accessibilityRole="header" style={styles.heading}>Профиль</Text>
      {state.status === "loading" && <Text accessibilityLiveRegion="polite">Загружаем профиль…</Text>}
      {state.status === "error" && <>
        <Text accessibilityRole="alert" style={styles.error}>Не удалось загрузить профиль.</Text>
        <Button title="Повторить загрузку" onPress={reload} />
      </>}
      {state.status === "ready" && <>
        <Text>{state.data.name}</Text>
        <Text>{profileRoleLabels[state.data.role]}</Text>
        <Text>{state.data.phone}</Text>
        <Text>Дата начала работы: {formatProfileDate(state.data.work_started_at)}</Text>
        <Text>Получать уведомления</Text>
        <Switch accessibilityLabel="Получать уведомления" value={state.data.notifications_enabled}
          disabled={saving} onValueChange={(enabled) => void save(enabled)} />
      </>}
      {saving && <Text accessibilityLiveRegion="polite">Сохраняем настройку…</Text>}
      {message ? <Text accessibilityRole="alert" style={styles.error}>{message}</Text> : null}
      <Button title="Выйти" onPress={() => onSessionEnd()} />
    </ScrollView>
  );
}
