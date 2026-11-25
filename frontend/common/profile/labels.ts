export const profileRoleLabels = {
  client: "Клиент",
  engineer: "Инженер",
  director: "Директор",
};

const profileDate = new Intl.DateTimeFormat("ru-RU", { dateStyle: "medium", timeZone: "Europe/Moscow" });

export function formatProfileDate(value?: string) {
  return value ? profileDate.format(new Date(value)) : "—";
}
