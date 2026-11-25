import { useRef, useState } from "react";
import { Button, Text, TextInput } from "react-native";
import { EngineerApiError } from "../../common/api/controller";
import { styles } from "../../common/styles";
import { loginEngineer } from "./controller";
import { readLoginForm } from "./form";

export function EngineerLogin({ sessionMessage, onSignIn }: {
  sessionMessage: string;
  onSignIn: (token: string, phone: string) => void;
}) {
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(sessionMessage);
  const submitting = useRef(false);

  function signIn() {
    if (submitting.current) return;

    const form = readLoginForm(phone, password);
    const body = form.body;
    if (!body) {
      setError(form.error);
      return;
    }

    submitting.current = true;
    setBusy(true);
    setError("");

    void loginEngineer(body).then((token) => {
      setPassword("");
      onSignIn(token, body.phone);
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
