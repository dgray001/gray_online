import {DEV} from "./util";

/** Data structure for all returns to get requests */
export interface ApiResponse<T> {
  success: boolean,
  result?: T,
  error_message?: string,
}

/** Returns websocket path */
export function websocketPath() {
  console.log('1');
  var scheme = window.location.protocol == "https:" ? 'wss://' : 'ws://';
  return DEV ? `ws://${location.hostname}:6807/api/lobby` :
    `${scheme}${location.hostname}${location.port ? ':' + location.port: ''}/api/lobby`;
}

/** Converts string api to actual api url */
function apiToUrl(api: string) {
  return `/api/${api}/`;
}

/** Calls and returns the input get api */
export async function apiGet<T>(api: string): Promise<ApiResponse<T>> {
  try {
    const response = await fetch(apiToUrl(api), {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json'
      },
    });
    return await response.json() as ApiResponse<T>;
  } catch(error) {
    console.log(error);
  }
  return {
    success: false,
    error_message: 'Server error; please report this bug.',
  };
}

/** Calls and returns the input post api */
export async function apiPost<T>(api: string, data: any): Promise<ApiResponse<T>> {
  const is_file = data instanceof File;
  const content_type = is_file ? data.type : 'application/json';
  const filename = is_file ? data.name : undefined;
  const body = is_file ? data : JSON.stringify(data);
  try {
    const response = await fetch(apiToUrl(api), {
      method: 'POST',
      headers: {
        'Content-Type': content_type,
        'X-File-Name': filename,
      },
      body,
    });
    return await response.json() as ApiResponse<T>;
  } catch(error) {
    console.log(error);
  }
  return {
    success: false,
    error_message: 'Server error; please report this bug.',
  };
}