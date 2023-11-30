
/** Calculates the modulus of two numbers */
export function modulus(n: number, m: number): number {
  return ((n % m) + m) % m;
}