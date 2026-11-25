"use client";

import { useQuery } from "@tanstack/react-query";
import { formatProfileDate, profileRoleLabels } from "@shared/profile/labels";
import { useSession } from "@/common/session/view";
import { Button } from "@/components/ui/button";
import { profileQuery, useNotifications } from "./controller";

export function ProfileScreen() {
  const { gateway } = useSession();
  const profile = useQuery(profileQuery(gateway));
  const notifications = useNotifications(gateway);

  return (
    <section className="mx-auto max-w-xl space-y-4">
      <h1>Профиль</h1>
      {profile.isPending && <p role="status">Загружаем профиль…</p>}
      {profile.isError && (
        <div className="space-y-2">
          <p role="alert">Не удалось загрузить профиль.</p>
          <Button variant="outline" onClick={() => void profile.refetch()}>Повторить</Button>
        </div>
      )}
      {profile.data && (
        <div className="space-y-4 rounded-xl border bg-card p-4 sm:p-5">
          <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2">
            <dt>Имя</dt><dd>{profile.data.name}</dd>
            <dt>Роль</dt><dd>{profileRoleLabels[profile.data.role]}</dd>
            <dt>Телефон</dt><dd>{profile.data.phone}</dd>
            <dt>{profile.data.role === "client" ? "Дата регистрации" : "Дата начала работы"}</dt>
            <dd>{formatProfileDate(profile.data.role === "client" ? profile.data.created_at : profile.data.work_started_at)}</dd>
          </dl>
          <label className="flex items-center gap-2">
            <input type="checkbox" checked={profile.data.notifications_enabled} className="size-4"
              disabled={notifications.isPending || profile.isError}
              onChange={({ currentTarget }) => notifications.mutate(currentTarget.checked)} />
            Получать уведомления
          </label>
          {notifications.isPending && <p role="status">Сохраняем настройку…</p>}
          {notifications.isError && <p role="alert">Не удалось подтвердить сохранение настройки.</p>}
        </div>
      )}
    </section>
  );
}
