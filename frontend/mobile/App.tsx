import { useState } from "react";
import { Button, FlatList, StatusBar, StyleSheet, Text, TextInput, View } from "react-native";
import type { components } from "../web/lib/api/schema";

type LoginRequest = components["schemas"]["LoginRequest"];
type TokenResponse = components["schemas"]["TokenResponse"];
type Booking = components["schemas"]["ServiceBookingListItem"];

const baseUrl = (process.env.EXPO_PUBLIC_API_URL ?? "http://10.0.2.2/api").replace(/\/+$/, "");

export default function App() {
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [bookings, setBookings] = useState<Booking[] | null>(null);

  async function signIn() {
    if (busy || !phone.trim() || !password) return;
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
      const pool = await fetch(`${baseUrl}/orders?status=requested`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!pool.ok) {
        setError(pool.status === 403 ? "Вход доступен только инженеру." : "Не удалось загрузить пул заявок.");
        return;
      }

      setBookings(await pool.json());
      setPassword("");
    } catch {
      setError("Не удалось связаться с сервером. Повторите попытку.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <View style={styles.screen}>
      <StatusBar barStyle="dark-content" />
      {bookings ? (
        <>
          <Text accessibilityRole="header" style={styles.heading}>Пул заявок</Text>
          <FlatList
            data={bookings}
            keyExtractor={(order) => order.id}
            renderItem={({ item }) => <Text style={styles.row}>{item.service || `Заявка ${item.id}`}</Text>}
            ListEmptyComponent={<Text>Открытых заявок пока нет.</Text>}
          />
          <Button title="Выйти" onPress={() => setBookings(null)} />
        </>
      ) : (
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
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, gap: 12, padding: 20, paddingTop: (StatusBar.currentHeight ?? 0) + 20, backgroundColor: "#f5f4f0" },
  heading: { fontSize: 24, fontWeight: "600", color: "#1a1a18" },
  input: { minHeight: 48, paddingHorizontal: 12, borderColor: "#b4b2a9", borderWidth: 1, borderRadius: 8, backgroundColor: "#fff" },
  row: { paddingVertical: 12 },
  error: { color: "#a32d2d" },
});
