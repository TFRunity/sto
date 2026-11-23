"use client";

import { useQuery } from "@tanstack/react-query";
import { useObjectUrl } from "@/common/hooks/use-object-url";
import { Button } from "@/components/ui/button";
import { useSession } from "@/common/session/view";
import { vehiclePhotoQuery } from "./controller";

export function VehicleCoverPhoto({ id }: { id: string }) {
  return <Photo id={id} label="Обложка автомобиля" className="max-h-48 w-full rounded-md object-contain" />;
}

export function Photo({ id, photoId, label, className = "max-h-[70vh] w-full rounded-md object-contain" }: {
  id: string;
  photoId?: string;
  label: string;
  className?: string;
}) {
  const { gateway } = useSession();
  const query = useQuery(vehiclePhotoQuery(gateway, id, photoId));
  const url = useObjectUrl(query.data);

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
