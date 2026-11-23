import { File } from "expo-file-system";
import type { operations } from "../../../../common/api/schema";
import { bearerHeader, gatewayRequest } from "../../../common/api/controller";

type PhotoUploadResponse = operations["addVehiclePhoto"]["responses"][201]["content"]["application/json"];

export function uploadVehiclePhoto(
  token: string, vehicleId: string, orderId: string, uri: string, signal: AbortSignal,
) {
  const body = new FormData();
  body.append("photo", new File(uri));
  body.append("order_id", orderId);

  return gatewayRequest(`/vehicles/${encodeURIComponent(vehicleId)}/photos`, {
    method: "POST", headers: bearerHeader(token), body, signal,
  }).then((response) => response.json() as Promise<PhotoUploadResponse>).then(({ photo_id }) => photo_id);
}
