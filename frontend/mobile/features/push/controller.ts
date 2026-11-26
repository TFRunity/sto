import * as Notifications from "expo-notifications";
import { useCallback, useEffect, useRef } from "react";
import { AppState } from "react-native";
import type { components } from "../../../common/api/schema";
import { bearerHeader, gatewayRequest } from "../../common/api/controller";

export type PushTarget = { kind: "booking" | "conversation"; id: string };

const projectId = process.env.EXPO_PUBLIC_EAS_PROJECT_ID;
const uuid = /^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/i;

if (projectId) {
  Notifications.setNotificationHandler({
    handleNotification: async () => ({
      shouldShowBanner: true,
      shouldShowList: true,
      shouldPlaySound: false,
      shouldSetBadge: false,
    }),
  });
}

export function usePushTarget(signedIn: boolean, open: (target: PushTarget) => void) {
  const response = Notifications.useLastNotificationResponse();

  useEffect(() => {
    if (!signedIn || !response) return;
    Notifications.clearLastNotificationResponse();
    if (response.actionIdentifier !== Notifications.DEFAULT_ACTION_IDENTIFIER) return;

    const { kind, order_id, dialog_id } = response.notification.request.content.data ?? {};
    if (kind === "booking_requested" && typeof order_id === "string" && uuid.test(order_id)) {
      open({ kind: "booking", id: order_id });
    }
    if (kind === "chat_message" && typeof dialog_id === "string" && uuid.test(dialog_id)) {
      open({ kind: "conversation", id: dialog_id });
    }
  }, [open, response, signedIn]);
}

async function getPushToken(
  id: string, notificationsEnabled: () => Promise<boolean>, signal: AbortSignal,
) {
  await Notifications.setNotificationChannelAsync("sto", {
    name: "Уведомления СТО", importance: Notifications.AndroidImportance.DEFAULT,
  });
  let permission = await Notifications.getPermissionsAsync();
  if (!permission.granted && permission.canAskAgain && await notificationsEnabled()) {
    if (signal.aborted) return;
    permission = await Notifications.requestPermissionsAsync();
  }
  if (!permission.granted || signal.aborted) return;
  return (await Notifications.getExpoPushTokenAsync({ projectId: id })).data;
}

async function updateRegistration(sessionToken: string, token: string, revoke: boolean) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 4000);
  try {
    const body: components["schemas"]["PushDeviceRegistration"] = { token };
    await gatewayRequest(revoke ? "/me/push-device/revoke" : "/me/push-device", {
      method: revoke ? "POST" : "PUT",
      headers: { ...bearerHeader(sessionToken), "Content-Type": "application/json" },
      body: JSON.stringify(body),
      signal: controller.signal,
    });
  } finally {
    clearTimeout(timeout);
  }
}

export function usePushDevice(
  sessionToken: string | undefined,
  preferenceRevision: number,
  notificationsEnabled: (token: string, signal: AbortSignal) => Promise<boolean>,
) {
  const queue = useRef(Promise.resolve());
  const registration = useRef<{ owner: string; token: string } | null>(null);
  const effect = useRef<AbortController | null>(null);

  useEffect(() => {
    if (!projectId || !sessionToken) return;
    const id = projectId;
    const controller = new AbortController();
    effect.current = controller;
    const { signal } = controller;

    async function sync(owner: string) {
      const token = await getPushToken(id, () => notificationsEnabled(owner, signal), signal);
      if (signal.aborted) return;
      queue.current = queue.current.then(async () => {
        if (signal.aborted) return;
        const previous = registration.current;
        if (previous?.owner === owner && previous.token !== token) {
          await updateRegistration(owner, previous.token, true);
          registration.current = null;
        }
        if (!token) return;
        registration.current = { owner, token };
        await updateRegistration(owner, token, false);
      }).catch(() => { /* Retry on the next application activation or preference change. */ });
    }

    let pending = false;
    function refresh() {
      if (pending || !sessionToken) return;
      pending = true;
      void sync(sessionToken)
        .catch(() => { /* Expo and the profile endpoint can be temporarily unavailable. */ })
        .finally(() => { pending = false; });
    }

    refresh();
    const active = AppState.addEventListener("change", (state) => { if (state === "active") refresh(); });
    const changed = Notifications.addPushTokenListener(refresh);
    return () => {
      controller.abort();
      active.remove();
      changed.remove();
    };
  }, [notificationsEnabled, preferenceRevision, sessionToken]);

  return useCallback(() => {
    effect.current?.abort();
    queue.current = queue.current.then(async () => {
      const device = registration.current;
      if (!device || device.owner !== sessionToken) return;
      registration.current = null;
      await updateRegistration(device.owner, device.token, true);
    }).catch(() => { /* An expired session or offline logout may prevent revocation. */ });
  }, [sessionToken]);
}
