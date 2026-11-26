import assert from "node:assert/strict";
import { setTimeout as delay } from "node:timers/promises";

const {
  API_URL, CHAT_CLIENT: client, CHAT_OUTSIDER: outsider, CHAT_ENGINEER: engineer,
  CHAT_SECOND_ENGINEER: secondEngineer, CHAT_VEHICLE_ID: vehicleId,
} = process.env;
const api = API_URL.replace(/\/+$/, "");
const sockets = new Set();

async function request(token, path, method = "GET", body, expected = 200) {
  const response = await fetch(api + path, {
    method,
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal: AbortSignal.timeout(10_000),
  });
  assert.equal(response.status, expected, `${method} ${path}`);
  const payload = await response.text();
  return payload ? JSON.parse(payload) : undefined;
}

async function connect(token) {
  const socket = new WebSocket(api.replace(/^http/, "ws") + "/chat/events", ["sto.chat.v1", token]);
  sockets.add(socket);
  const events = [];
  socket.addEventListener("message", ({ data }) => events.push(JSON.parse(data)));
  socket.addEventListener("close", () => sockets.delete(socket));
  await new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error("WebSocket handshake timed out")), 5000);
    socket.addEventListener("open", () => { clearTimeout(timer); resolve(); }, { once: true });
    socket.addEventListener("error", () => { clearTimeout(timer); reject(new Error("WebSocket handshake failed")); }, { once: true });
  });
  assert.equal(socket.protocol, "sto.chat.v1");
  return { socket, events };
}

async function expectEvent(stream, dialogId, type) {
  const deadline = Date.now() + 5000;
  while (Date.now() < deadline) {
    const index = stream.events.findIndex((event) => event.dialog_id === dialogId && event.type === type);
    if (index >= 0) {
      assert.deepEqual(stream.events.splice(index, 1)[0], { type, dialog_id: dialogId });
      return;
    }
    await delay(20);
  }
  assert.fail(`Missing ${type} event for dialog ${dialogId}`);
}

const openOrderDialog = (orderId) => request(client, "/chat/dialogs/order", "POST", { order_id: orderId });
const messages = (id) => `/chat/dialogs/${id}/messages`;
const send = (token, id, body) => request(token, messages(id), "POST", { body }, 201);

async function checkChat() {
  const date = new Date(Date.now() + 86_400_000).toISOString().slice(0, 10);
  const { order_id: orderId } = await request(client, "/orders", "POST", {
    vehicle_id: vehicleId, service: "Проверка WebSocket", date,
  }, 201);
  await request(engineer, `/orders/${orderId}/claim`, "PATCH");
  const { dialog_id: original } = await openOrderDialog(orderId);
  const [clientStream, engineerStream, outsiderStream, secondStream] = await Promise.all(
    [client, engineer, outsider, secondEngineer].map(connect),
  );

  const first = await send(client, original, "Первое сообщение");
  await expectEvent(engineerStream, original, "message");
  await expectEvent(clientStream, original, "message");
  const read = () => request(engineer, `/chat/dialogs/${original}/read`, "PATCH", {
    through_message_id: first.id,
  }, 204);
  await read();
  await expectEvent(clientStream, original, "read");
  await expectEvent(engineerStream, original, "read");
  await read();

  await request(secondEngineer, `/orders/${orderId}/claim`, "PATCH");
  await request(engineer, messages(original), "GET", undefined, 404);
  const archived = (await request(client, "/chat/dialogs")).find(({ id }) => id === original);
  assert.equal(archived.can_write, false);
  await request(client, messages(original), "POST", { body: "Архив" }, 403);
  const { dialog_id: current } = await openOrderDialog(orderId);
  assert.notEqual(current, original);
  await send(client, current, "Сообщение новому инженеру");
  await expectEvent(secondStream, current, "message");
  await expectEvent(clientStream, current, "message");

  await new Promise((resolve) => {
    clientStream.socket.addEventListener("close", resolve, { once: true });
    clientStream.socket.close();
  });
  const missed = await send(secondEngineer, current, "Сообщение во время обрыва");
  await expectEvent(secondStream, current, "message");
  const reconnected = await connect(client);
  const history = await request(client, messages(current));
  assert(history.some(({ id, is_read }) => id === missed.id && !is_read));
  const dialogs = await request(client, "/chat/dialogs");
  assert.equal(dialogs.find(({ id }) => id === current).unread_count, 1);
  await send(secondEngineer, current, "Сообщение после восстановления");
  await expectEvent(reconnected, current, "message");

  // Прежний инженер остался подключён, но события нового диалога ему недоступны.
  assert.equal(engineerStream.socket.readyState, WebSocket.OPEN);
  assert.equal(outsiderStream.socket.readyState, WebSocket.OPEN);
  assert.deepEqual(engineerStream.events, []);
  assert.deepEqual(outsiderStream.events, []);
  console.log("Chat WebSocket checked: auth, delivery, receipts, reassignment and HTTP recovery");
}

try {
  await checkChat();
} finally {
  for (const socket of sockets) socket.close();
}
