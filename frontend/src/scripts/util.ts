/** Whether in DEV environment; used for public api keys */
export const DEV = true;

/** Loop helper function */
export const loop = (times: number, callback: Function) => {
  for (let i = 0; i < times; i++) {
    callback(i);
  }
};

/** Async loop helper function */
export const asyncLoop = async (times: number, callback: Function) => {
  return new Promise<void>(async (resolve) => {
    for (let i = 0; i < times; i++) {
      await callback(i);
    }
    resolve();
  });
};

/** Returns promise that resolves when condition function becomes true */
export function until(condition: () => boolean, poll_timer = 300) {
  const poll = (resolve: () => void) => {
    if (condition()) {
      resolve();
    }
    else {
      setTimeout(() => poll(resolve), poll_timer);
    }
  }
  return new Promise<void>(poll);
}

/** Capitalizes each word in string */
export function capitalize(str: string): string {
  const str_split = str.trim().split(' ');
  return str_split.map(str => {
    const lower = str.toLowerCase();
    return str.charAt(0).toUpperCase() + lower.slice(1);
  }).join(' ');
}