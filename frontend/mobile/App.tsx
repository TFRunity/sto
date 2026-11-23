import { useCallback, useEffect, useRef, useState } from "react";
import { Button, FlatList, StatusBar, StyleSheet, Text, TextInput, View } from "react-native";
import { normalizeRussianPhone } from "../common/phone/utils";
import { EngineerApiError, loadBookingPool, loginEngineer, type Booking } from "./engineer-api";

export default function App() {
  const [token, setToken] = useState<string | null>(null);
  const [sessionMessage, setSessionMessage] = useState("");
  const endSession = useCallback((message = "") => {
    setToken(null);
    setSessionMessage(message);
  }, []);

  return (
    <View style={styles.screen}>
      <StatusBar barStyle="dark-content" />
      {token
        ? <BookingPool token={token} onSessionEnd={endSession} />
        : <EngineerLogin sessionMessage={sessionMessage} onSignIn={setToken} />}
    </View>
  );
}

function EngineerLogin({ sessionMessage, onSignIn }: {
  sessionMessage: string;
  onSignIn: (token: string) => void;
}) {
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(sessionMessage);
  const submitting = useRef(false);

  function signIn() {
    if (submitting.current) return;

    const normalizedPhone = normalizeRussianPhone(phone);
    if (!normalizedPhone) {
      setError("Введите полный номер телефона.");
      return;
    }
    if (!password) {
      setError("Введите пароль.");
      return;
    }

    submitting.current = true;
    setBusy(true);
    setError("");

    void loginEngineer({ phone: normalizedPhone, password }).then((token) => {
      setPassword("");
      onSignIn(token);
    }).catch((cause: unknown) => {
      setError(cause instanceof EngineerApiError && cause.status === 401
        ? "Проверьте телефон и пароль."
        : "Не удалось выполнить вход. Повторите попытку.");
    }).finally(() => {
      submitting.current = false;
      setBusy(false);
    });
  }

  return (
    <>
      <Text accessibilityRole="header" style={styles.heading}>Вход инженера</Text>
      <Text>Телефон</Text>
      <TextInput style={styles.input} accessibilityLabel="Телефон" autoComplete="tel"
        keyboardType="phone-pad" value={phone} onChangeText={setPhone} editable={!busy} />
      <Text>Пароль</Text>
      <TextInput style={styles.input} accessibilityLabel="Пароль" autoComplete="current-password"
        secureTextEntry value={password} onChangeText={setPassword} editable={!busy} />
      {error ? <Text accessibilityRole="alert" style={styles.error}>{error}</Text> : null}
      <Button title={busy ? "Входим…" : "Войти"} disabled={busy || !phone.trim() || !password} onPress={signIn} />
    </>
  );
}

type PoolState =
  | { status: "loading" }
  | { status: "error" }
  | { status: "ready"; bookings: Booking[] };

function BookingPool({ token, onSessionEnd }: {
  token: string;
  onSessionEnd: (message?: string) => void;
}) {
  const [state, setState] = useState<PoolState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    const controller = new AbortController();

    void loadBookingPool(token, controller.signal).then((bookings) => {
      if (!controller.signal.aborted) setState({ status: "ready", bookings });
    }).catch((cause: unknown) => {
      if (controller.signal.aborted) return;
      if (cause instanceof EngineerApiError && cause.status === 401) {
        onSessionEnd("Сессия истекла. Войдите снова.");
        return;
      }
      if (cause instanceof EngineerApiError && cause.status === 403) {
        onSessionEnd("Вход доступен только инженеру.");
        return;
      }

      setState({ status: "error" });
    });

    return () => controller.abort();
  }, [attempt, onSessionEnd, token]);

  return (
    <>
      <View style={styles.header}>
        <Text accessibilityRole="header" style={styles.heading}>Пул заявок</Text>
        <Button title="Выйти" onPress={() => onSessionEnd()} />
      </View>
      {state.status === "loading" && <Text accessibilityLiveRegion="polite">Загружаем заявки…</Text>}
      {state.status === "error" && (
        <>
          <Text accessibilityRole="alert" style={styles.error}>Не удалось загрузить пул заявок.</Text>
          <Button title="Повторить загрузку" onPress={() => {
            setState({ status: "loading" });
            setAttempt((value) => value + 1);
          }} />
        </>
      )}
      {state.status === "ready" && (
        <FlatList
          data={state.bookings}
          keyExtractor={(order) => order.id}
          renderItem={({ item }) => <Text style={styles.row}>{item.service || `Заявка ${item.id}`}</Text>}
          ListEmptyComponent={<Text>Открытых заявок пока нет.</Text>}
        />
      )}
    </>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, gap: 12, padding: 20, paddingTop: (StatusBar.currentHeight ?? 0) + 20, backgroundColor: "#f5f4f0" },
  header: { flexDirection: "row", flexWrap: "wrap", justifyContent: "space-between", alignItems: "center", gap: 8 },
  heading: { fontSize: 24, fontWeight: "600", color: "#1a1a18" },
  input: { minHeight: 48, paddingHorizontal: 12, borderColor: "#b4b2a9", borderWidth: 1, borderRadius: 8, backgroundColor: "#fff" },
  row: { paddingVertical: 12 },
  error: { color: "#a32d2d" },
});
