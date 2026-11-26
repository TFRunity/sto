"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useRef, useState, type FormEvent } from "react";
import { canSendMessage, dialogTitle, lastUnreadMessage, type ChatDialog } from "@shared/chat/dialog";
import { cabinetPath, type WebRole } from "@/common/session/session";
import { useSession } from "@/common/session/view";
import { Button } from "@/components/ui/button";
import { GatewayError } from "@/lib/api/client";
import { chatDialogsQuery, chatMessagesQuery, useChatActions, useChatEvents, useOpenChat, type ChatTarget } from "./controller";
import { messageField } from "./form";

export function ChatStartButton({ target, children }: { target: ChatTarget; children: string }) {
  const { gateway, session } = useSession();
  const open = useOpenChat(gateway);
  const router = useRouter();
  return (
    <div className="space-y-1">
      <Button variant="outline" disabled={open.isPending} onClick={() => open.mutate(target, {
        onSuccess: (result) => {
          if (session && result) router.push(dialogPath(session.role, result.dialog_id));
        },
      })}>{children}</Button>
      {open.isError && <p role="alert">
        {open.error instanceof GatewayError && open.error.code === "order_engineer_not_assigned"
          ? "Чат откроется после назначения ответственного инженера."
          : "Не удалось открыть диалог. Попробуйте снова."}
      </p>}
    </div>
  );
}

export function ChatScreen({ role }: { role: WebRole }) {
  return <Suspense fallback={<p role="status">Загружаем чат…</p>}><ChatWorkspace role={role} /></Suspense>;
}

function ChatWorkspace({ role }: { role: WebRole }) {
  const { session, gateway } = useSession();
  const selectedId = useSearchParams().get("dialog");
  const dialogs = useQuery(chatDialogsQuery(gateway));
  const selected = dialogs.data?.find((dialog) => dialog.id === selectedId);
  useChatEvents(session?.token);
  return (
    <section className="mx-auto max-w-5xl space-y-4">
      <h1>Чат</h1>
      {role === "client" && <ChatStartButton target={{ kind: "support" }}>Написать в поддержку</ChatStartButton>}
      {dialogs.isPending && <p role="status">Загружаем диалоги…</p>}
      {dialogs.isError && (
        <div>
          <p role="alert">Не удалось загрузить диалоги.</p>
          <Button variant="outline" onClick={() => void dialogs.refetch()}>Повторить</Button>
        </div>
      )}
      {dialogs.isSuccess && (
        <div className="grid gap-4 md:grid-cols-[minmax(14rem,1fr)_2fr]">
          <nav aria-label="Диалоги" className="rounded-xl border bg-card p-3">
            {!dialogs.data.length && <p className="p-2">Диалогов пока нет.</p>}
            {dialogs.data.map((dialog) => (
              <Link key={dialog.id} href={dialogPath(role, dialog.id)}
                aria-current={selectedId === dialog.id ? "page" : undefined}
                className="flex justify-between gap-3 rounded-md px-3 py-2 hover:bg-muted aria-[current=page]:bg-muted">
                {dialogTitle(dialog, role)}
                {dialog.unread_count > 0 && <span aria-label={`Непрочитано: ${dialog.unread_count}`}>{dialog.unread_count}</span>}
              </Link>
            ))}
          </nav>
          {selected
            ? <Conversation key={selected.id} dialog={selected} role={role} />
            : <p>{selectedId ? "Диалог недоступен." : "Выберите диалог."}</p>}
        </div>
      )}
    </section>
  );
}

function Conversation({ dialog, role }: { dialog: ChatDialog; role: WebRole }) {
  const { gateway } = useSession();
  const messages = useQuery(chatMessagesQuery(gateway, dialog.id));
  const { send, markRead } = useChatActions(gateway, dialog.id);
  const [body, setBody] = useState("");
  const history = useRef<HTMLOListElement>(null);
  const attemptedRead = useRef<string | undefined>(undefined);
  const unreadId = messages.isSuccess ? lastUnreadMessage(messages.data, dialog.peer_id)?.id : undefined;
  const latestId = messages.isSuccess ? messages.data.at(-1)?.id : undefined;
  const mark = markRead.mutate;

  useEffect(() => {
    if (latestId && history.current) history.current.scrollTop = history.current.scrollHeight;
  }, [latestId]);

  useEffect(() => {
    if (!dialog.can_write || !unreadId) return;
    const id = unreadId;
    function markVisible() {
      if (document.visibilityState !== "visible" || attemptedRead.current === id) return;
      attemptedRead.current = id;
      mark(id);
    }
    // В фоновой вкладке просмотр начнётся после возвращения к диалогу.
    markVisible();
    document.addEventListener("visibilitychange", markVisible);
    return () => document.removeEventListener("visibilitychange", markVisible);
  }, [dialog.can_write, mark, unreadId]);

  function submit(event: FormEvent) {
    event.preventDefault();
    if (send.isPending || !canSendMessage(body)) return;
    send.mutate(body.trim(), { onSuccess: () => setBody("") });
  }

  return (
    <section className="min-w-0 space-y-3 rounded-xl border bg-card p-4">
      <h2>{dialogTitle(dialog, role)}</h2>
      {messages.isPending && <p role="status">Загружаем историю…</p>}
      {messages.isError && (
        <div>
          <p role="alert">Не удалось загрузить историю.</p>
          <Button variant="outline" onClick={() => void messages.refetch()}>Повторить</Button>
        </div>
      )}
      {messages.isSuccess && (
        <>
          {!messages.data.length && <p>Сообщений пока нет.</p>}
          <ol ref={history} aria-label="Сообщения" className="max-h-[50vh] space-y-2 overflow-y-auto">
            {messages.data.map((message) => {
              const own = message.sender_id !== dialog.peer_id;
              return (
                <li key={message.id} className={`w-fit max-w-[90%] rounded-lg px-3 py-2 ${own ? "ml-auto bg-muted" : "border"}`}>
                  <p className="whitespace-pre-wrap break-words">{message.body}</p>
                  {own && <p className="text-xs text-muted-foreground">{message.is_read ? "Прочитано" : "Отправлено"}</p>}
                </li>
              );
            })}
          </ol>
          {dialog.can_write
            ? <form className="space-y-2" onSubmit={submit}>
                <label htmlFor="chat-message" className="sr-only">{messageField.label}</label>
                <textarea id="chat-message" placeholder={messageField.placeholder} value={body}
                  onChange={(event) => setBody(event.target.value)} disabled={send.isPending}
                  className="min-h-20 w-full rounded-md border bg-background p-2" />
                {send.isError && <p role="alert">Не удалось отправить сообщение.</p>}
                <Button disabled={send.isPending || !canSendMessage(body)}>Отправить</Button>
                {markRead.isError && unreadId && (
                  <Button type="button" variant="outline" onClick={() => mark(unreadId)}>Повторить отметку прочтения</Button>
                )}
              </form>
            : <p>Диалог доступен для чтения.</p>}
        </>
      )}
    </section>
  );
}

function dialogPath(role: WebRole, id: string) {
  return `${cabinetPath[role]}/chat?dialog=${encodeURIComponent(id)}`;
}
