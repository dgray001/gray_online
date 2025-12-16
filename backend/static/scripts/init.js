// @ts-check
'use strict';
export { };

// Get a new version each hour, so caching can still be effective
const { version } = await import(`/scripts/version.js?v=${Math.floor(Date.now() / (1000 * 60 * 60))}`);

// Access the script element and read the data attribute
const scriptUrl = new URL(import.meta.url);
const page = scriptUrl.searchParams.get('page');

const params = new URL(window.location.href).searchParams;
params.set('v', version);
window.history.replaceState(null, '', `?${params.toString()}`);
const script = document.createElement('script');
script.setAttribute('src', `/dist/${page}.bundle.js?v=${version}`);
document.head.appendChild(script);
