function formatQuery(params: URLSearchParams): string {
  return Array.from(params, ([key, value]: [string, string]): string => {
    const encoded = encodeURIComponent(value);
    const readable =
      key === 'game_settings' ? encoded.replace(/%(7B|7D|22|3A|2C|5B|5D)/g, decodeURIComponent) : encoded;
    return `${encodeURIComponent(key)}=${readable}`;
  }).join('&');
}

export function getUrlParam(key: string): string {
  const url = new URL(window.location.href);
  return url.searchParams.get(key) ?? '';
}

export function setUrlParam(key: string, value: string): void {
  const params = new URL(window.location.href).searchParams;
  if (!!value) {
    params.set(key, value);
  } else {
    params.delete(key);
  }
  window.history.replaceState(null, '', `?${formatQuery(params)}${window.location.hash}`);
}

export function removeUrlParam(key: string): void {
  const params = new URL(window.location.href).searchParams;
  params.delete(key);
  window.history.replaceState(null, '', `?${formatQuery(params)}${window.location.hash}`);
}

export function getPage(): string {
  const url = new URL(window.location.href);
  return url.pathname;
}

/** Navigates to the input page */
export function navigate(page: string): void {
  if (!page.startsWith('./') && !page.startsWith('../')) {
    page = './' + page;
  }
  const new_url = new URL(page, location.protocol + '//' + location.host);
  const url = new URL(window.location.href);
  for (const [key, value] of url.searchParams.entries()) {
    new_url.searchParams.set(key, value);
  }
  new_url.search = formatQuery(new_url.searchParams);
  window.location.assign(new_url);
}
