import type { components } from "../api/schema";

export type ChatEvent = components["schemas"]["ChatEvent"];

export function subscribeToChat(
  apiUrl: string,
  token: string,
  onConnect: () => void,
  onEvent: (event: ChatEvent) => void,
) {
  const url = apiUrl.replace(/^http/, "ws") + "/chat/events";
  let socket: WebSocket;
  let retry: ReturnType<typeof setTimeout> | undefined;
  let retryDelay = 500;
  let stopped = false;

  function connect() {
    if (stopped) return;
    const current = new WebSocket(url, ["sto.chat.v1", token]);
    socket = current;
    current.onopen = () => {
      if (stopped) return;
      retryDelay = 500;
      onConnect();
    };
    current.onmessage = ({ data }) => {
      if (stopped || typeof data !== "string") return;
      try {
        const event: ChatEvent = JSON.parse(data);
        if ((event.type === "message" || event.type === "read") && typeof event.dialog_id === "string") {
          onEvent(event);
        }
      } catch {
        // Игнорируем повреждённое событие. Состояние хранится в HTTP API.
      }
    };
    current.onclose = () => {
      if (stopped) return;
      retry = setTimeout(connect, retryDelay);
      retryDelay = Math.min(retryDelay * 2, 15_000);
    };
  }

  connect();
  return () => {
    stopped = true;
    clearTimeout(retry);
    socket.close();
  };
}
