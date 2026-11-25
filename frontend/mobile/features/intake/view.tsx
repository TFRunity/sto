import { useEffect, useRef, useState } from "react";
import { Button, ScrollView, Text, TextInput } from "react-native";
import { EngineerApiError, sessionErrorMessage } from "../../common/api/controller";
import { styles } from "../../common/styles";
import { openIntakeAttempt } from "./attempt";
import { createIntakeVehicle, createOrder } from "./controller";
import { prepareVehicle, type IntakeClient, type IntakeVehicle } from "./form";
import { findVehicle, searchClients } from "./search/controller";

export function IntakeScreen({ token, engineerPhone, onBack, onOrderOpened, onSessionEnd }: {
  token: string;
  engineerPhone: string;
  onBack: () => void;
  onOrderOpened: (id: string) => void;
  onSessionEnd: (message: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<IntakeClient[]>();
  const [client, setClient] = useState<IntakeClient>();
  const [number, setNumber] = useState("");
  const [vehicle, setVehicle] = useState<IntakeVehicle>();
  const [missingVehicle, setMissingVehicle] = useState(false);
  const [brand, setBrand] = useState("");
  const [vin, setVin] = useState("");
  const [error, setError] = useState("");
  const [pending, setPending] = useState(false);
  const request = useRef<AbortController | null>(null);

  useEffect(() => () => request.current?.abort(), []);

  async function run(action: (signal: AbortSignal) => Promise<void>, message: string) {
    if (request.current) return;
    const controller = new AbortController();
    request.current = controller;
    setError("");
    setPending(true);
    try {
      await action(controller.signal);
    } catch (cause: unknown) {
      if (controller.signal.aborted) return;
      const sessionError = sessionErrorMessage(cause);
      if (sessionError) onSessionEnd(sessionError);
      else setError(message);
    } finally {
      request.current = null;
      if (!controller.signal.aborted) setPending(false);
    }
  }

  function pickClient(value: IntakeClient) {
    setClient(value);
    setResults(undefined);
  }

  function lookUpClient() {
    setResults(undefined);
    void run(async (signal) => {
      try {
        const matches = await searchClients(token, query.trim(), signal);
        if (!signal.aborted) setResults(matches);
      } catch (cause: unknown) {
        if (cause instanceof EngineerApiError && cause.status === 422) {
          setError("Слишком много совпадений. Уточните имя или телефон.");
          return;
        }
        throw cause;
      }
    }, "Не удалось найти клиента. Попробуйте снова.");
  }

  function lookUpVehicle() {
    setVehicle(undefined);
    setMissingVehicle(false);
    void run(async (signal) => {
      try {
        const match = await findVehicle(token, number.trim(), signal);
        if (!signal.aborted) setVehicle(match);
      } catch (cause: unknown) {
        if (cause instanceof EngineerApiError && cause.status === 404) {
          setMissingVehicle(true);
          return;
        }
        throw cause;
      }
    }, "Не удалось проверить госномер. Попробуйте снова.");
  }

  async function ensureVehicle(ownerId: string, signal: AbortSignal) {
    if (vehicle) return vehicle;
    try {
      const created = await createIntakeVehicle(token, prepareVehicle(ownerId, number, brand, vin), signal);
      if (signal.aborted) return;
      // Карточка сохраняется отдельным запросом. Если создание заказа не подтвердилось, используем её при повторе.
      setVehicle(created);
      setMissingVehicle(false);
      return created;
    } catch (cause: unknown) {
      if (signal.aborted) return;
      if (cause instanceof EngineerApiError && cause.status === 409) {
        setMissingVehicle(false);
        setError("Этот госномер уже занят. Найдите ТС по номеру ещё раз.");
        return;
      }
      throw cause;
    }
  }

  function submit() {
    if (!client || !canSubmit) return;
    void run(async (signal) => {
      const selected = await ensureVehicle(client.id, signal);
      if (!selected || signal.aborted) return;
      const attempt = openIntakeAttempt(engineerPhone, client.id, selected.vehicle_id);
      const order = await createOrder(token, {
        client_id: client.id, vehicle_id: selected.vehicle_id, request_id: attempt.key,
      }, signal);
      if (signal.aborted) return;
      attempt.file.delete();
      onOrderOpened(order.order_id);
    }, "Не удалось подтвердить создание заказа. Повторите отправку для той же машины и клиента.");
  }

  const ownerMismatch = Boolean(vehicle && client && vehicle.owner_id !== client.id);
  const canSubmit = client && !ownerMismatch && (vehicle || (missingVehicle && brand.trim() && vin.trim()));

  return (
    <ScrollView keyboardShouldPersistTaps="handled" contentContainerStyle={{ gap: 10 }}>
      <Button title="К пулу заявок" onPress={onBack} disabled={pending} />
      <Text accessibilityRole="header" style={styles.heading}>Прямой приём</Text>
      <Text>Клиент</Text>
      {client ? (
        <>
          <Text>{client.name} · {client.phone}</Text>
          <Button title="Выбрать другого клиента" onPress={() => setClient(undefined)} disabled={pending} />
        </>
      ) : (
        <>
          <TextInput accessibilityLabel="Имя или телефон клиента" placeholder="Имя или телефон"
            style={styles.input} value={query} maxLength={200} editable={!pending}
            onChangeText={(value) => { setQuery(value); setResults(undefined); }} />
          <Button title="Найти клиента" onPress={lookUpClient} disabled={pending || query.trim().length < 2} />
          {results?.map((match) => (
            <Button key={match.id} title={`${match.name} · ${match.phone}`} onPress={() => pickClient(match)} />
          ))}
          {results?.length === 0 && <Text>Клиент не найден. Попросите его зарегистрироваться в вебе.</Text>}
        </>
      )}
      <Text>Госномер ТС</Text>
      <TextInput accessibilityLabel="Госномер ТС" placeholder="Введите госномер" autoCapitalize="characters"
        style={styles.input} value={number} maxLength={32} editable={!pending}
        onChangeText={(value) => { setNumber(value); setVehicle(undefined); setMissingVehicle(false); }} />
      <Button title="Найти ТС" onPress={lookUpVehicle} disabled={pending || !number.trim()} />
      {vehicle && <Text>{vehicle.brand} · {vehicle.number}</Text>}
      {ownerMismatch && <Text accessibilityRole="alert" style={styles.error}>ТС принадлежит другому клиенту.</Text>}
      {missingVehicle && (
        <>
          <Text>ТС с таким номером не найдено. Заполните карточку нового ТС.</Text>
          <TextInput accessibilityLabel="Марка ТС" placeholder="Марка" style={styles.input}
            value={brand} onChangeText={setBrand} editable={!pending} />
          <TextInput accessibilityLabel="VIN" placeholder="VIN" style={styles.input} autoCapitalize="characters"
            value={vin} onChangeText={setVin} editable={!pending} />
        </>
      )}
      {error ? <Text accessibilityRole="alert" style={styles.error}>{error}</Text> : null}
      {pending && <Text accessibilityLiveRegion="polite">Выполняем запрос…</Text>}
      <Button title="Создать заказ" disabled={pending || !canSubmit} onPress={submit} />
    </ScrollView>
  );
}
