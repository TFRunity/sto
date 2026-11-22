"use client";

import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { useSession } from "@/features/auth/session-provider";
import type { components } from "@/lib/api/schema";
import { vehiclePhotoQuery } from "./customer-api";

export function VehiclePhotos({ vehicle }: { vehicle: components["schemas"]["VehicleCard"] }) {
  return (
    <section className="space-y-3 rounded-xl border bg-card p-4 sm:p-5">
      <h2 className="text-lg font-semibold">Фотофиксация</h2>
      {vehicle.cover_photo_url && <PhotoDialog id={vehicle.vehicle_id} label="Обложка автомобиля" />}
      <div className="flex flex-wrap gap-2">
        {vehicle.photos.map((photo, index) => (
          <PhotoDialog key={photo.photo_id} id={vehicle.vehicle_id} photoId={photo.photo_id} label={`Снимок ${index + 1}`} />
        ))}
      </div>
      {!vehicle.cover_photo_url && !vehicle.photos.length && <p>Фотографий пока нет.</p>}
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

function Photo({ id, photoId, label }: { id: string; photoId?: string; label: string }) {
  const { gateway } = useSession();
  const query = useQuery(vehiclePhotoQuery(gateway, id, photoId));
  const [url, setUrl] = useState<string>();

  useEffect(() => {
    if (!query.data) return;
    // Снимок загружается с токеном, поэтому для img создаём локальный URL из Blob.
    const objectUrl = URL.createObjectURL(query.data);
    setUrl(objectUrl);
    return () => URL.revokeObjectURL(objectUrl);
  }, [query.data]);

  if (query.isError) return (
    <div className="space-y-2">
      <p role="alert">Фотография недоступна.</p>
      <Button variant="outline" onClick={() => void query.refetch()}>Повторить</Button>
    </div>
  );

  return url
    ? <img className="max-h-[70vh] w-full rounded-md object-contain" src={url} alt={label} />
    : <p role="status">Загружаем фотографию…</p>;
}
