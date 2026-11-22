"use client";

import { useEffect, useState } from "react";

export function usePhotoUrl(photo: Blob | undefined) {
  const [preview, setPreview] = useState<{ photo: Blob; url: string }>();

  useEffect(() => {
    if (!photo) return;

    // Локальный URL нужен img для файла или защищённого ответа Gateway.
    const url = URL.createObjectURL(photo);
    setPreview({ photo, url });
    return () => URL.revokeObjectURL(url);
  }, [photo]);

  return preview && preview.photo === photo ? preview.url : undefined;
}
