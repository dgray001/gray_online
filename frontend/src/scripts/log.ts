const THROTTLE_MS = 1500;
const MAX_TRACKED_MESSAGES = 200;
const last_logged = new Map<string, number>();

/** Returns whether this exact message hasn't been logged in the last 1.5s */
function shouldLog(args: unknown[]): boolean {
  const key = args.map((arg) => String(arg)).join('|');
  const now = Date.now();
  const last = last_logged.get(key);
  if (last !== undefined && now - last < THROTTLE_MS) {
    return false;
  }
  if (last_logged.size >= MAX_TRACKED_MESSAGES) {
    last_logged.clear();
  }
  last_logged.set(key, now);
  return true;
}

export function log(...args: unknown[]): void {
  if (shouldLog(args)) {
    console.log(...args);
  }
}

export function warn(...args: unknown[]): void {
  if (shouldLog(args)) {
    console.warn(...args);
  }
}

export function err(...args: unknown[]): void {
  if (shouldLog(args)) {
    console.error(...args);
  }
}
