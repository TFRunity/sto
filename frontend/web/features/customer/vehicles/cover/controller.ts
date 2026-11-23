import type { QueryClient } from "@tanstack/react-query";
import type { GatewayClient } from "@/lib/api/client";
import { vehiclePhotoQuery, vehicleQuery } from "../../shared/vehicles/controller";

export function replaceVehicleCover(gateway: GatewayClient, queryClient: QueryClient, id: string, photo: File) {
  return gateway.PUT("/vehicles/{id}/cover_photo", {
    params: { path: { id } },
    body: { photo: photo.name },
    bodySerializer: () => {
      // OpenAPI описывает бинарное поле как string. В multipart передаём сам файл.
      const form = new FormData();
      form.set("photo", photo);
      return form;
    },
  }).then(() => Promise.all([
    queryClient.invalidateQueries({ queryKey: vehicleQuery(gateway, id).queryKey, exact: true }),
    queryClient.invalidateQueries({ queryKey: vehiclePhotoQuery(gateway, id).queryKey, exact: true }),
  ]));
}
