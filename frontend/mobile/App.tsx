import { useCallback, useEffect, useRef, useState } from "react";
import { Button, FlatList, StatusBar, StyleSheet, Text, TextInput, View } from "react-native";
import type { components } from "../web/lib/api/schema";

type LoginRequest = components["schemas"]["LoginRequest"];
type TokenResponse = components["schemas"]["TokenResponse"];
type Booking = components["schemas"]["ServiceBookingListItem"];

const baseUrl = (process.env.EXPO_PUBLIC_API_URL ?? "http://10.0.2.2/api").replace(/\/+$/, "");

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

  async function signIn() {
    if (submitting.current || !phone.trim() || !password) return;
    submitting.current = true;
    setBusy(true);
    setError("");
    try {
      const body: LoginRequest = { phone: phone.trim(), password };
      const response = await fetch(`${baseUrl}/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) {
        setError(response.status === 401 ? "Проверьте телефон и пароль." : "Не удалось выполнить вход.");
        return;
      }

      const { token }: TokenResponse = await response.json();
      if (!token) throw new Error("Missing access token");
      setPassword("");
      onSignIn(token);
    } catch {
      setError("Не удалось выполнить вход. Повторите попытку.");
    } finally {
      submitting.current = false;
      setBusy(false);
    }
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

    async function load() {
      try {
        const response = await fetch(`${baseUrl}/orders?status=requested`, {
          headers: { Authorization: `Bearer ${token}` },
          signal: controller.signal,
        });
        if (controller.signal.aborted) return;
        if (response.status === 401 || response.status === 403) {
          onSessionEnd(response.status === 401 ? "Сессия истекла. Войдите снова." : "Вход доступен только инженеру.");
          return;
        }
        if (!response.ok) throw new Error("Booking pool request failed");
        const bookings: Booking[] = await response.json();
        if (!controller.signal.aborted) setState({ status: "ready", bookings });
      } catch {
        if (!controller.signal.aborted) setState({ status: "error" });
      }
    }

    void load();
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
