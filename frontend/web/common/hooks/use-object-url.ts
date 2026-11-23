"use client";

import { useEffect, useState } from "react";

export function useObjectUrl(blob: Blob | undefined) {
  const [objectUrl, setObjectUrl] = useState<{ blob: Blob; url: string }>();

  useEffect(() => {
    if (!blob) return;

    const url = URL.createObjectURL(blob);
    setObjectUrl({ blob, url });
    return () => URL.revokeObjectURL(url);
  }, [blob]);

  return objectUrl && objectUrl.blob === blob ? objectUrl.url : undefined;
}
