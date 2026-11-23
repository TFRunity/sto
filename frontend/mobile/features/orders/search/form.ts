import { createSearchMatcher } from "../../../../common/search/utils";

export type PositionKind = "item" | "service";

export function filterPositions<T extends { name: string }>(entries: readonly T[], input: string) {
  const matches = createSearchMatcher(input, "substring");
  return entries.filter(({ name }) => matches(name));
}
