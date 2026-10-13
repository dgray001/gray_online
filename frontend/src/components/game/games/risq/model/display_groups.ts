export function displayGroups<T>(
  entities: readonly T[],
  capacity: number,
  id_key: (entity: T) => string,
  category_key?: (entity: T) => string
): T[][] {
  if (entities.length <= capacity) {
    return entities.map((entity: T): T[] => [entity]);
  }
  let result: T[][] = [];
  for (const key of [id_key, category_key]) {
    if (!key) {
      break;
    }
    const groups = new Map<string, T[]>();
    for (const entity of entities) {
      const name = key(entity);
      const group = groups.get(name) ?? [];
      group.push(entity);
      groups.set(name, group);
    }
    result = [...groups.values()];
    if (result.length <= capacity) {
      break;
    }
  }
  return result;
}
