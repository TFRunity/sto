import { useCallback, useEffect, useRef, useState } from "react";
import { Button, FlatList, StyleSheet, Text, TextInput, View } from "react-native";
import { canSendMessage, dialogTitle, lastUnreadMessage } from "../../../common/chat/dialog";
import { styles } from "../../common/styles";
import { chatSessionExpired, loadConversation, loadDialogs, markRead, sendMessage, useChatData } from "./controller";
import { messageField } from "./form";

type ScreenProps = {
  token: string;
  onBack: () => void;
  onSessionEnd: (message: string) => void;
};

export function ChatList({ token, onBack, onSessionEnd, onOpen }: ScreenProps & { onOpen: (id: string) => void }) {
  const load = useCallback((signal: AbortSignal) => loadDialogs(token, signal), [token]);
  const { data, error, refresh } = useChatData(token, undefined, load, onSessionEnd);
  return (
    <>
      <ChatHeader title="Чат" onBack={onBack} />
      {!data && !error && <Text>Загружаем диалоги…</Text>}
      {!!error && <LoadError message="Не удалось загрузить диалоги." onRetry={refresh} />}
      {data && (
        <FlatList data={data} keyExtractor={(dialog) => dialog.id}
          ListEmptyComponent={<Text>Диалогов пока нет.</Text>}
          renderItem={({ item }) => (
            <Button title={`${dialogTitle(item, "engineer")}${item.unread_count ? ` · Непрочитано: ${item.unread_count}` : ""}`}
              onPress={() => onOpen(item.id)} />
          )} />
      )}
    </>
  );
}

export function ChatConversation({ token, id, onBack, onSessionEnd }: ScreenProps & { id: string }) {
  const load = useCallback((signal: AbortSignal) => loadConversation(token, id, signal), [id, token]);
  const { data, error, active, refresh } = useChatData(token, id, load, onSessionEnd);
  const [body, setBody] = useState("");
  const [sending, setSending] = useState(false);
  const [sendFailed, setSendFailed] = useState(false);
  const [readFailed, setReadFailed] = useState(false);
  const attemptedRead = useRef<string | undefined>(undefined);
  const unreadId = data ? lastUnreadMessage(data.messages, data.dialog.peer_id)?.id : undefined;

  const acknowledge = useCallback((messageId: string) => {
    attemptedRead.current = messageId;
    setReadFailed(false);
    void markRead(token, id, messageId).then(refresh).catch((failure: unknown) => {
      if (!chatSessionExpired(failure, onSessionEnd)) setReadFailed(true);
    });
  }, [id, onSessionEnd, refresh, token]);

  useEffect(() => {
    if (!active || !data?.dialog.can_write || !unreadId || attemptedRead.current === unreadId) return;
    acknowledge(unreadId);
  }, [acknowledge, active, data?.dialog.can_write, unreadId]);

  function submit() {
    if (!data?.dialog.can_write || sending || !canSendMessage(body)) return;
    setSending(true);
    setSendFailed(false);
    void sendMessage(token, id, body.trim()).then(() => setBody("")).catch((failure: unknown) => {
      if (!chatSessionExpired(failure, onSessionEnd)) setSendFailed(true);
    }).finally(() => {
      setSending(false);
      refresh();
    });
  }

  return (
    <>
      <ChatHeader title={data ? dialogTitle(data.dialog, "engineer") : "Диалог"} onBack={onBack} />
      {data === undefined && !error && <Text>Загружаем историю…</Text>}
      {!!error && <LoadError message="Не удалось загрузить историю." onRetry={refresh} />}
      {data === null && <Text>Диалог недоступен.</Text>}
      {data && (
        <>
          <FlatList inverted={data.messages.length > 0} data={[...data.messages].reverse()} keyExtractor={(message) => message.id}
            ListEmptyComponent={<Text>Сообщений пока нет.</Text>}
            renderItem={({ item }) => {
              const own = item.sender_id !== data.dialog.peer_id;
              return (
                <View style={[chatStyles.message, own && chatStyles.ownMessage]}>
                  <Text>{item.body}</Text>
                  {own && <Text style={chatStyles.receipt}>{item.is_read ? "Прочитано" : "Отправлено"}</Text>}
                </View>
              );
            }} />
          {data.dialog.can_write ? (
            <>
              <TextInput accessibilityLabel={messageField.label} placeholder={messageField.placeholder} value={body}
                multiline editable={!sending} onChangeText={setBody} style={chatStyles.input} />
              {sendFailed && <Text accessibilityRole="alert" style={styles.error}>Не удалось отправить сообщение.</Text>}
              <Button title="Отправить" disabled={sending || !canSendMessage(body)} onPress={submit} />
              {readFailed && unreadId && <Button title="Повторить отметку прочтения" onPress={() => acknowledge(unreadId)} />}
            </>
          ) : <Text>Диалог доступен для чтения.</Text>}
        </>
      )}
    </>
  );
}

function ChatHeader({ title, onBack }: { title: string; onBack: () => void }) {
  return (
    <View style={styles.header}>
      <Text accessibilityRole="header" style={styles.heading}>{title}</Text>
      <Button title="Назад" onPress={onBack} />
    </View>
  );
}

function LoadError({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <>
      <Text accessibilityRole="alert" style={styles.error}>{message}</Text>
      <Button title="Повторить" onPress={onRetry} />
    </>
  );
}

const chatStyles = StyleSheet.create({
  message: { alignSelf: "flex-start", backgroundColor: "#f1f5f9", borderRadius: 8, marginVertical: 4, maxWidth: "90%", padding: 10 },
  ownMessage: { alignSelf: "flex-end", backgroundColor: "#dbeafe" },
  receipt: { color: "#475569", fontSize: 12 },
  input: { borderColor: "#94a3b8", borderRadius: 8, borderWidth: 1, maxHeight: 120, padding: 8 },
});
