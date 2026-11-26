import { useCallback, useEffect, useState } from "react";
import { AppState } from "react-native";
import type { ChatDialog, ChatMessage } from "../../../common/chat/dialog";
import { subscribeToChat } from "../../../common/chat/events";
import { bearerHeader, EngineerApiError, gatewayBaseUrl, gatewayRequest, sessionErrorMessage } from "../../common/api/controller";

export function loadDialogs(token: string, signal: AbortSignal): Promise<ChatDialog[]> {
  return gatewayRequest("/chat/dialogs", { headers: bearerHeader(token), signal }).then((response) => response.json());
}

export async function loadConversation(token: string, id: string, signal: AbortSignal) {
  const dialogs = await loadDialogs(token, signal);
  const dialog = dialogs.find((item) => item.id === id);
  if (!dialog) return null;
  const response = await gatewayRequest(`/chat/dialogs/${id}/messages`, { headers: bearerHeader(token), signal });
  const messages: ChatMessage[] = await response.json();
  return { dialog, messages };
}

export function sendMessage(token: string, id: string, body: string) {
  return changeDialog(token, `/chat/dialogs/${id}/messages`, "POST", { body });
}

export function markRead(token: string, id: string, messageId: string) {
  return changeDialog(token, `/chat/dialogs/${id}/read`, "PATCH", { through_message_id: messageId });
}

function changeDialog(token: string, path: string, method: string, body: Record<string, string>) {
  return gatewayRequest(path, {
    method, headers: { ...bearerHeader(token), "Content-Type": "application/json" }, body: JSON.stringify(body),
  });
}

export function chatSessionExpired(error: unknown, onSessionEnd: (message: string) => void) {
  if (!(error instanceof EngineerApiError) || error.status !== 401) return false;
  onSessionEnd(sessionErrorMessage(error) ?? "Войдите снова.");
  return true;
}

export function useChatData<T>(
  token: string,
  dialogId: string | undefined,
  load: (signal: AbortSignal) => Promise<T>,
  onSessionEnd: (message: string) => void,
) {
  const [active, setActive] = useState(!AppState.currentState || AppState.currentState === "active");
  const [revision, setRevision] = useState(0);
  const [state, setState] = useState<{ data?: T; error?: unknown }>({});
  const refresh = useCallback(() => setRevision((value) => value + 1), []);

  useEffect(() => {
    const subscription = AppState.addEventListener("change", (status) => setActive(status === "active"));
    return () => subscription.remove();
  }, []);

  useEffect(() => {
    if (!active) return;
    return subscribeToChat(gatewayBaseUrl, token, refresh, (event) => {
      if (!dialogId || dialogId === event.dialog_id) refresh();
    });
  }, [active, dialogId, refresh, token]);

  useEffect(() => {
    if (!active) return;
    const controller = new AbortController();
    void load(controller.signal).then((data) => {
      if (!controller.signal.aborted) setState({ data });
    }).catch((error: unknown) => {
      if (!controller.signal.aborted && !chatSessionExpired(error, onSessionEnd)) setState({ error });
    });
    return () => controller.abort();
  }, [active, load, onSessionEnd, revision]);

  return { ...state, active, refresh };
}
