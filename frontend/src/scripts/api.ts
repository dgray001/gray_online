import { DEV } from './util';

interface ApiSuccessResponse<T> {
  success: true;
  result: T;
  error_message?: never;
}

interface ApiErrorResponse {
  success: false;
  result?: never;
  error_message: string;
}

/** Data structure for all returns to get requests */
export type ApiResponse<T> = ApiSuccessResponse<T> | ApiErrorResponse;

/** Returns websocket path */
export function websocketPath() {
  const scheme = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
  return DEV
    ? `ws://${location.hostname}:6807/api/lobby`
    : `${scheme}${location.hostname}${location.port ? ':' + location.port : ''}/api/lobby`;
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
        'Content-Type': 'application/json',
      },
    });
    return (await response.json()) as ApiResponse<T>;
  } catch (error) {
    console.log(error);
  }
  return {
    success: false,
    error_message: 'Server error; please report this bug.',
  };
}

/** Calls and returns the input post api */
export async function apiPost<T>(api: string, data: unknown): Promise<ApiResponse<T>> {
  const is_file = data instanceof File;
  const content_type = is_file ? data.type : 'application/json';
  const filename = is_file ? data.name : undefined;
  const body = is_file ? data : JSON.stringify(data);
  try {
    const custom_headers = new Headers();
    custom_headers.append('Content-Type', content_type);
    custom_headers.append('X-File-Name', filename ?? '');
    const response = await fetch(apiToUrl(api), {
      method: 'POST',
      headers: custom_headers,
      body,
    });
    return (await response.json()) as ApiResponse<T>;
  } catch (error) {
    console.log(error);
  }
  return {
    success: false,
    error_message: 'Server error; please report this bug.',
  };
}
