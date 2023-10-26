export function getUrlParam(key: string): string {
  const url = new URL(window.location.href);
  return url.searchParams.get(key) ?? '';
}

export function setUrlParam(key: string, value: string): void {
  const params = new URL(window.location.href).searchParams;
  if (!!value) {
    params.set(key, value);
  }
  else {
    params.delete(key);
  }
  window.history.replaceState(null, '', `?${params.toString()}`);
}

export function removeUrlParam(key: string): void {
  const params = new URL(window.location.href).searchParams;
  params.delete(key);
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
  window.location.assign(new_url);
}