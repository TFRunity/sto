import { createSearchMatcher } from "@shared/search/utils";
import type { VehicleMake } from "../controller";

export function matchMakes(makes: VehicleMake[], input: string) {
  const matches = createSearchMatcher(input, "prefix-with-aliases");
  return makes.filter(({ name, aliases }) => matches(name, aliases)).slice(0, 20);
}

export function findMake(makes: VehicleMake[], input: string) {
  const matches = createSearchMatcher(input, "exact");
  return makes.find(({ name }) => matches(name));
}
