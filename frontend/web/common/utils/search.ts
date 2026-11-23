type SearchMode = "exact" | "prefix" | "prefix-with-aliases" | "substring";

function searchKey(value: string) {
  return value.trim().toLowerCase().normalize("NFD").replace(/\p{M}/gu, "");
}

export function createSearchMatcher(input: string, mode: SearchMode) {
  const query = searchKey(input);

  return (name: string, aliases: readonly string[] = []) => {
    const key = searchKey(name);

    switch (mode) {
      case "exact":
        return key === query;
      case "prefix":
        return key.startsWith(query);
      case "prefix-with-aliases":
        return key.startsWith(query) || aliases.some((alias) => searchKey(alias).startsWith(query));
      case "substring":
        return key.includes(query);
    }
  };
}
