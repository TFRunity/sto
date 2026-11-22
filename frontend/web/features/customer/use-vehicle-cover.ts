"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { useSession } from "@/features/auth/session-provider";
import { GatewayError } from "@/lib/api/client";
import { CoverImageError, prepareCoverImage } from "./cover-image";
import { replaceVehicleCover, vehiclePhotoQuery, vehicleQuery } from "./customer-api";

export function useVehicleCover(id: string, onSaved: () => void) {
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
      .then((photo) => replaceVehicleCover(gateway, id, photo))
      .then(() => Promise.all([
        queryClient.invalidateQueries({ queryKey: vehicleQuery(gateway, id).queryKey, exact: true }),
        queryClient.invalidateQueries({ queryKey: vehiclePhotoQuery(gateway, id).queryKey, exact: true }),
      ]))
      .then(onSaved)
      .catch((cause: unknown) => {
        if (cause instanceof CoverImageError) {
          setError(cause.message);
        } else if (cause instanceof GatewayError && cause.status === 413) {
          setError("Файл слишком большой. Выберите изображение до 3 МБ.");
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
