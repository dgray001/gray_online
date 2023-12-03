
/** Calculates the modulus of two numbers */
export function modulus(n: number, m: number): number {
  return ((n % m) + m) % m;
}

/** Calculates 'correct' atan in [0, 2pi) with the jump at 0 */
export function atangent(y: number, x: number): number {
  const atan = Math.atan2(y, x);
  if (atan === 0) {
    return 0; // to prevent the completely ridiculous "-0" JS BS
  }
  if (atan < 0) {
    return -atan;
  }
  return 2 * Math.PI - atan;
}
