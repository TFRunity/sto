"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { useObjectUrl } from "@/common/hooks/use-object-url";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { VehicleCoverPhoto } from "../../shared/vehicles/view";
import { ImagePreparationError } from "@/common/utils/image";
import { useSession } from "@/common/session/view";
import { GatewayError } from "@/lib/api/client";
import { replaceVehicleCover } from "./controller";
import { coverImageAccept, coverImageFormatHint, coverImageSizeHint, prepareCoverImage } from "./form";

function CoverEditor({ id, onSaved }: { id: string; onSaved: () => void }) {
  const { file, pending, error, selectFile, save } = useVehicleCover(id, onSaved);
  const preview = useObjectUrl(file);

  return (
    <DialogContent
      showCloseButton={!pending}
      onEscapeKeyDown={(event) => { if (pending) event.preventDefault(); }}
      onInteractOutside={(event) => { if (pending) event.preventDefault(); }}
    >
      <DialogTitle>Обложка автомобиля</DialogTitle>
      <DialogDescription>{coverImageFormatHint}</DialogDescription>
      <form className="space-y-4" aria-busy={pending} onSubmit={(event) => {
        event.preventDefault();
        save();
      }}>
        <fieldset disabled={pending} className="space-y-3">
          <div className="space-y-1.5">
            <label htmlFor="cover-photo" className="text-sm font-medium">Файл</label>
            <Input id="cover-photo" type="file" accept={coverImageAccept}
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

export function VehicleCover({ id, hasCover }: { id: string; hasCover: boolean }) {
  const [open, setOpen] = useState(false);

  return (
    <div className="space-y-3">
      {hasCover && <VehicleCoverPhoto id={id} />}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger asChild>
          <Button variant="outline">{hasCover ? "Заменить обложку" : "Добавить обложку"}</Button>
        </DialogTrigger>
        {open && <CoverEditor id={id} onSaved={() => setOpen(false)} />}
      </Dialog>
    </div>
  );
}

function useVehicleCover(id: string, onSaved: () => void) {
  const { gateway } = useSession();
  const queryClient = useQueryClient();
  const uploading = useRef(false);
  const [file, setFile] = useState<File>();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");

  function selectFile(next: File | undefined) {
    setFile(next);
    setError("");
  }

  function save() {
    if (!file || uploading.current) return;
    uploading.current = true;
    setPending(true);
    setError("");

    void prepareCoverImage(file)
      .then((photo) => replaceVehicleCover(gateway, queryClient, id, photo))
      .then(onSaved)
      .catch((cause: unknown) => {
        if (cause instanceof ImagePreparationError) {
          setError(cause.message);
        } else if (cause instanceof GatewayError && cause.status === 413) {
          setError(`Файл слишком большой. ${coverImageSizeHint}`);
        } else {
          setError("Не удалось сохранить обложку. Попробуйте ещё раз.");
        }
      })
      .finally(() => {
        uploading.current = false;
        setPending(false);
      });
  }

  return { file, pending, error, selectFile, save };
}
