/** Whether in DEV environment; used for public api keys */
export const DEV = true;

/** Loop helper function */
export const loop = (times: number, callback: (i?: number) => void) => {
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
export async function untilTimer(timer: number): Promise<void> {
  await new Promise(resolve => setTimeout(resolve, timer));
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
export function createLock(throwaway_extras = false) {
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
      if (!throwaway_extras) {
        queue.push(runFn);
      }
    } else {
      running = true;
      runFn();
    }
    return delayed_promise;
  };
}

/** Downloads input string as file */
export function download(content: string, fileName: string, contentType = 'text/plain') {
  var a = document.createElement("a");
  var file = new Blob([content], {type: contentType});
  a.href = URL.createObjectURL(file);
  a.download = fileName;
  a.click();
}

/** Detects whether the current browser is a mobile browser */
export function clientOnMobile() {
  const regex1 = /(android|bb\d+|meego).+mobile|avantgo|bada\/|blackberry|blazer|compal|elaine|fennec|hiptop|iemobile|ip(hone|od)|iris|kindle|lge |maemo|midp|mmp|mobile.+firefox|netfront|opera m(ob|in)i|palm( os)?|phone|p(ixi|re)\/|plucker|pocket|psp|series(4|6)0|symbian|treo|up\.(browser|link)|vodafone|wap|windows ce|xda|xiino/i;
  const regex2 = /1207|6310|6590|3gso|4thp|50[1-6]i|770s|802s|a wa|abac|ac(er|oo|s\-)|ai(ko|rn)|al(av|ca|co)|amoi|an(ex|ny|yw)|aptu|ar(ch|go)|as(te|us)|attw|au(di|\-m|r |s )|avan|be(ck|ll|nq)|bi(lb|rd)|bl(ac|az)|br(e|v)w|bumb|bw\-(n|u)|c55\/|capi|ccwa|cdm\-|cell|chtm|cldc|cmd\-|co(mp|nd)|craw|da(it|ll|ng)|dbte|dc\-s|devi|dica|dmob|do(c|p)o|ds(12|\-d)|el(49|ai)|em(l2|ul)|er(ic|k0)|esl8|ez([4-7]0|os|wa|ze)|fetc|fly(\-|_)|g1 u|g560|gene|gf\-5|g\-mo|go(\.w|od)|gr(ad|un)|haie|hcit|hd\-(m|p|t)|hei\-|hi(pt|ta)|hp( i|ip)|hs\-c|ht(c(\-| |_|a|g|p|s|t)|tp)|hu(aw|tc)|i\-(20|go|ma)|i230|iac( |\-|\/)|ibro|idea|ig01|ikom|im1k|inno|ipaq|iris|ja(t|v)a|jbro|jemu|jigs|kddi|keji|kgt( |\/)|klon|kpt |kwc\-|kyo(c|k)|le(no|xi)|lg( g|\/(k|l|u)|50|54|\-[a-w])|libw|lynx|m1\-w|m3ga|m50\/|ma(te|ui|xo)|mc(01|21|ca)|m\-cr|me(rc|ri)|mi(o8|oa|ts)|mmef|mo(01|02|bi|de|do|t(\-| |o|v)|zz)|mt(50|p1|v )|mwbp|mywa|n10[0-2]|n20[2-3]|n30(0|2)|n50(0|2|5)|n7(0(0|1)|10)|ne((c|m)\-|on|tf|wf|wg|wt)|nok(6|i)|nzph|o2im|op(ti|wv)|oran|owg1|p800|pan(a|d|t)|pdxg|pg(13|\-([1-8]|c))|phil|pire|pl(ay|uc)|pn\-2|po(ck|rt|se)|prox|psio|pt\-g|qa\-a|qc(07|12|21|32|60|\-[2-7]|i\-)|qtek|r380|r600|raks|rim9|ro(ve|zo)|s55\/|sa(ge|ma|mm|ms|ny|va)|sc(01|h\-|oo|p\-)|sdk\/|se(c(\-|0|1)|47|mc|nd|ri)|sgh\-|shar|sie(\-|m)|sk\-0|sl(45|id)|sm(al|ar|b3|it|t5)|so(ft|ny)|sp(01|h\-|v\-|v )|sy(01|mb)|t2(18|50)|t6(00|10|18)|ta(gt|lk)|tcl\-|tdg\-|tel(i|m)|tim\-|t\-mo|to(pl|sh)|ts(70|m\-|m3|m5)|tx\-9|up(\.b|g1|si)|utst|v400|v750|veri|vi(rg|te)|vk(40|5[0-3]|\-v)|vm40|voda|vulc|vx(52|53|60|61|70|80|81|83|85|98)|w3c(\-| )|webc|whit|wi(g |nc|nw)|wmlb|wonu|x700|yas\-|your|zeto|zte\-/i;
  return regex1.test(navigator.userAgent) || regex2.test(navigator.userAgent.substring(0, 4));
}

/** Copies input text to clipboard */
export async function copyToClipboard(text: string): Promise<boolean> {
  const alternateCopy: () => boolean = () => {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.top = '0';
    textarea.style.left = '0';
    textarea.style.position = 'fixed';
    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();
    let success = false;
    try {
      success = document.execCommand('copy');
    } catch(e) {}
    document.body.removeChild(textarea);
    return success;
  };
  if (!navigator.clipboard) {
    return alternateCopy();
  }
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch(e) {
    return alternateCopy();
  }
}
