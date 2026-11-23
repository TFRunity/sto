"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import type { components } from "@shared/api/schema";
import { Photo } from "../../shared/vehicles/view";

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
