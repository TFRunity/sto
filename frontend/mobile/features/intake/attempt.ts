import { File, Paths } from "expo-file-system";
import { uuid } from "expo-modules-core";

export function openIntakeAttempt(engineerPhone: string, clientId: string, vehicleId: string) {
  const name = uuid.v5(`sto/intake/${engineerPhone}/${clientId}/${vehicleId}`, uuid.namespace.dns);
  const file = new File(Paths.document, `intake-${name}.txt`);
  if (file.exists) {
    const key = file.textSync().trim();
    if (key) return { file, key };
  }
  const key = uuid.v4();
  // Запись ключа должна завершиться до POST. После потери ответа повтор переживает выход из формы и перезапуск приложения.
  if (!file.exists) file.create();
  file.write(key);
  return { file, key };
}
