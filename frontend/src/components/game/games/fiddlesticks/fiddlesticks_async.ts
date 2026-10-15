import { until } from '../../../../scripts/util';
import { getSettings } from '../../../../scripts/settings_store';

export function animationDuration(duration: number): number {
  return duration * getSettings().fiddlesticks.animation_speed;
}

export function waitForAnimation(duration: number, signal: AbortSignal): Promise<void> {
  return waitForDelay(animationDuration(duration), signal);
}

export async function waitFor(condition: () => boolean, signal: AbortSignal): Promise<void> {
  await until(condition, 20, signal);
  signal.throwIfAborted();
}

export function waitForDelay(duration: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve, reject): void => {
    signal.throwIfAborted();
    const timer = setTimeout((): void => {
      signal.removeEventListener('abort', aborted);
      resolve();
    }, duration);
    function aborted(): void {
      clearTimeout(timer);
      reject(signal.reason);
    }
    signal.addEventListener('abort', aborted, { once: true });
  });
}
