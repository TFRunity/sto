import type { VehicleMake } from "./customer-api";

function searchKey(value: string) {
  return value.trim().toLowerCase().normalize("NFD").replace(/\p{M}/gu, "");
}

export function matchMakes(makes: VehicleMake[], input: string) {
  const prefix = searchKey(input);

  return makes.filter(({ name, aliases }) =>
    searchKey(name).startsWith(prefix) || aliases.some((alias) => searchKey(alias).startsWith(prefix)),
  ).slice(0, 20);
}

export function findMake(makes: VehicleMake[], input: string) {
  const key = searchKey(input);
  return makes.find(({ name }) => searchKey(name) === key);
}
