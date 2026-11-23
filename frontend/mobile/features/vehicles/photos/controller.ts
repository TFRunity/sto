import { bearerHeader, gatewayRequest } from "../../../common/api/controller";

export type VehiclePhoto = { kind: "cover" } | { kind: "gallery"; id: string };

function readPhotoDataUrl(blob: Blob) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result === "string") resolve(reader.result);
      else reject(new Error("Missing vehicle photo"));
    };
    reader.onerror = () => reject(reader.error ?? new Error("Could not read vehicle photo"));
    reader.readAsDataURL(blob);
  });
}

export async function loadVehiclePhoto(token: string, vehicleId: string, photo: VehiclePhoto, signal: AbortSignal) {
  const resource = photo.kind === "cover" ? "cover_photo" : `photos/${encodeURIComponent(photo.id)}`;
  const response = await gatewayRequest(`/vehicles/${encodeURIComponent(vehicleId)}/${resource}`, {
    headers: { ...bearerHeader(token), "Cache-Control": "no-cache" }, signal,
  });

  // React Native holds server blobs in native memory until they are closed.
  const blob: Blob & { close?: () => void } = await response.blob();
  try {
    return await readPhotoDataUrl(blob);
  } finally {
    blob.close?.();
  }
}
