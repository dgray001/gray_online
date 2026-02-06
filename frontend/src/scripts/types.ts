/** Allows making specific properties optional */
export type PartialBy<T, K extends keyof T> = Omit<T, K> & Partial<Pick<T, K>>;

/** Allows specifying specific properties to be required */
export type WithRequired<T, K extends keyof T> = T & { [P in K]-?: T[P] };
