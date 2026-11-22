"use client";

import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { useSession } from "@/features/auth/session-provider";
import type { components } from "@/lib/api/schema";
import { vehiclePhotoQuery } from "./customer-api";
import { usePhotoUrl } from "./use-photo-url";
import { useVehicleCover } from "./use-vehicle-cover";

export function VehicleCover({ id, hasCover }: { id: string; hasCover: boolean }) {
  const [open, setOpen] = useState(false);

  return (
    <div className="space-y-3">
      {hasCover && <Photo id={id} label="Обложка автомобиля" className="max-h-48 w-full rounded-md object-contain" />}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button variant="outline">{hasCover ? "Заменить обложку" : "Добавить обложку"}</Button>
        </DialogTrigger>
        {open && <CoverEditor id={id} onSaved={() => setOpen(false)} />}
      </Dialog>
    </div>
  );
}

function CoverEditor({ id, onSaved }: { id: string; onSaved: () => void }) {
  const { file, pending, error, selectFile, save } = useVehicleCover(id, onSaved);
  const preview = usePhotoUrl(file);

  return (
    <DialogContent
      showCloseButton={!pending}
      onEscapeKeyDown={(event) => { if (pending) event.preventDefault(); }}
      onInteractOutside={(event) => { if (pending) event.preventDefault(); }}
    >
      <DialogTitle>Обложка автомобиля</DialogTitle>
      <DialogDescription>Выберите фото в формате JPEG или PNG.</DialogDescription>
      <form className="space-y-4" aria-busy={pending} onSubmit={(event) => {
        event.preventDefault();
        save();
      }}>
        <fieldset disabled={pending} className="space-y-3">
          <div className="space-y-1.5">
            <label htmlFor="cover-photo" className="text-sm font-medium">Файл</label>
            <Input id="cover-photo" type="file" accept="image/jpeg,image/png"
              onChange={(event) => selectFile(event.currentTarget.files?.[0])} />
          </div>
          {preview && <img src={preview} alt="Предпросмотр обложки" className="max-h-52 w-full rounded-md object-contain" />}
          {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
          <Button type="submit" disabled={!file}>{pending ? "Сохраняем…" : "Сохранить обложку"}</Button>
        </fieldset>
      </form>
    </DialogContent>
  );
}

export function VehiclePhotos({ vehicle }: { vehicle: components["schemas"]["VehicleCard"] }) {
  return (
    <section className="space-y-3 rounded-xl border bg-card p-4 sm:p-5">
      <h2 className="text-lg font-semibold">Фотофиксация</h2>
      <div className="flex flex-wrap gap-2">
        {vehicle.photos.map((photo, index) => (
          <PhotoDialog key={photo.photo_id} id={vehicle.vehicle_id} photoId={photo.photo_id} label={`Снимок ${index + 1}`} />
        ))}
      </div>
      {!vehicle.photos.length && <p>Снимков фотофиксации пока нет.</p>}
    </section>
  );
}

function PhotoDialog({ id, photoId, label }: { id: string; photoId?: string; label: string }) {
  const [open, setOpen] = useState(false);
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild><Button variant="outline">{label}</Button></DialogTrigger>
      <DialogContent aria-describedby={undefined}>
        <DialogTitle>{label}</DialogTitle>
        {open && <Photo id={id} photoId={photoId} label={label} />}
      </DialogContent>
    </Dialog>
  );
}

function Photo({ id, photoId, label, className = "max-h-[70vh] w-full rounded-md object-contain" }: {
  id: string;
  photoId?: string;
  label: string;
  className?: string;
}) {
  const { gateway } = useSession();
  const query = useQuery(vehiclePhotoQuery(gateway, id, photoId));
  const url = usePhotoUrl(query.data);

  if (query.isError) return (
    <div className="space-y-2">
      <p role="alert">Фотография недоступна.</p>
      <Button variant="outline" onClick={() => void query.refetch()}>Повторить</Button>
    </div>
  );

  return url
    ? <img className={className} src={url} alt={label} />
    : <p role="status">Загружаем фотографию…</p>;
}
