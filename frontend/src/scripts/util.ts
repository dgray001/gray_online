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
export function until(condition: () => boolean, poll_timer = 300): Promise<void> {
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

/** Awaits a specific amount of time */
export function untilTimer(timer: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, timer));
}

/** Capitalizes each word in string */
export function capitalize(str: string): string {
  const str_split = str.trim().split(' ');
  return str_split.map(str => {
    const lower = str.toLowerCase();
    return str.charAt(0).toUpperCase() + lower.slice(1);
  }).join(' ');
}

/** Options for clickButton function */
interface ClickButtonOptions {
  loading_text?: string;
  disable_button?: boolean;
  re_enable_button?: boolean;
}

type clickButtonReturn = string|void;

/** Handles boilerplate from clicking button */
export function clickButton(
  button: HTMLButtonElement,
  fn: () => clickButtonReturn|Promise<clickButtonReturn>,
  input_options: ClickButtonOptions = {},
) {
  const options: ClickButtonOptions = {...{
    loading_text: undefined,
    disable_button: true,
    re_enable_button: true,
  }, ...input_options};
  button.addEventListener('click', () => {
    if (options.disable_button) {
      button.disabled = true;
    }
    let has_changed_text = false;
    const original_text = button.innerText;
    function changeButtonText(text?: clickButtonReturn, enable_button?: boolean) {
      if (typeof text === 'string') {
        button.innerText = text;
      } else if (has_changed_text) {
        button.innerText = original_text;
      }
      if (enable_button) {
        button.disabled = false;
      }
    }
    changeButtonText(options.loading_text);
    const result = fn();
    if (result instanceof Promise) {
      result.then((completed_text) => {
        changeButtonText(completed_text, options.re_enable_button);
      });
    } else {
      changeButtonText(result, options.re_enable_button);
    }
  });
}

/** Returns enum keys as a list */
export function enumKeys<O extends object, K extends keyof O = keyof O>(obj: O): K[] {
  return Object.keys(obj).filter(k => Number.isNaN(parseInt(k))) as K[];
}

/** Sets interval x times */
export function setIntervalX(
  fn: (counter?: number) => void, delay: number, repetitions: number, fn_end = () => {}
): NodeJS.Timer {
  let counter = 0;
  const intervalId = setInterval(() => {
    counter++;
    if (counter > repetitions) {
      clearInterval(intervalId);
      fn_end();
    } else {
      fn(counter);
    }
  }, delay);
  return intervalId;
}

/** Returns a method lock */
export function createLock() {
  let running = false;
  const queue: Array<() => Promise<unknown>> = [];
  return (fn: () => Promise<unknown>) => {
    let delayed_resolve: (value: unknown) => void;
    let delayed_reject: (value: unknown) => void;
    const delayed_promise = new Promise((resolve, reject) => {
      delayed_resolve = resolve;
      delayed_reject = reject;
    });
    const runFn = async () => {
      await fn().then(delayed_resolve, delayed_reject);
      const first = queue.shift();
      if (first) {
        await first();
      } else {
        running = false;
      }
    };
    if (running) {
      queue.push(runFn);
    } else {
      running = true;
      runFn();
    }
    return delayed_promise;
  };
}
