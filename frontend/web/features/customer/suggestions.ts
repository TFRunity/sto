import { createSearchMatcher } from "@/common/utils/search";
import type { VehicleMake } from "./vehicle-api";

export function matchMakes(makes: VehicleMake[], input: string) {
  const matches = createSearchMatcher(input, "prefix-with-aliases");
  return makes.filter(({ name, aliases }) => matches(name, aliases)).slice(0, 20);
}

export function matchServiceNames(names: string[], input: string) {
  const matches = createSearchMatcher(input, "substring");
  return names.filter((name) => matches(name)).slice(0, 20);
}

export function findMake(makes: VehicleMake[], input: string) {
  const matches = createSearchMatcher(input, "exact");
  return makes.find(({ name }) => matches(name));
}
