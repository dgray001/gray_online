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

/** Clamps number that may or may not be undefined */
export function clampNumber(n: number, min: number, max: number, default_min = true): number {
  if (isNaN(min) || min === undefined || min === null) {
    return n;
  }
  if (isNaN(max) || max === undefined || max === null) {
    return n;
  }
  if (max < min) {
    return clampNumber(n, max, min);
  }
  if (isNaN(n) || n === undefined || n === null) {
    return default_min ? min : max;
  }
  if (n > max) {
    return max;
  }
  if (n < min) {
    return min;
  }
  return n;
}

/** Bounded number */
export declare interface BoundedNumber {
  value_min: number;
  value_max: number;
  value: number;
}

/** Returns valid bounded number */
export function validateBoundedNumber(n: BoundedNumber): BoundedNumber {
  if (isNaN(n.value)) {
    n.value = 0;
  }
  if (isNaN(n.value_min)) {
    n.value_min = 0;
  }
  if (isNaN(n.value_max)) {
    n.value_max = 0;
  }
  if (n.value_max < n.value_min) {
    const max = n.value_max;
    n.value_max = n.value_min;
    n.value_min = max;
  }
  if (n.value < n.value_min) {
    n.value = n.value_min;
  }
  if (n.value > n.value_max) {
    n.value = n.value_max;
  }
  return n;
}

/** Sets a new value for a bounded number, returning the set value */
export function setBoundedNumber(n: BoundedNumber, v: number): number {
  if (isNaN(v)) {
    return n.value;
  }
  if (v < n.value_min) {
    n.value = n.value_min;
  } else if (v > n.value_max) {
    n.value = n.value_max;
  } else {
    n.value = v;
  }
  return n.value;
}
