/**
 * Makes all properties in T and its nested objects optional, recursively.
 * Handles objects and arrays. Primitives and functions remain unchanged.
 */
export type DeepPartial<T> = T extends Function
  ? T // Keep functions as-is.
  : T extends Array<infer U>
    ? DeepPartial<U>[] // Recurse into array elements.
    : T extends object
      ? {
          // Recurse into object properties.
          [P in keyof T]?: DeepPartial<T[P]>; // Make property optional and recurse.
        }
      : T; // Keep primitives as-is.
