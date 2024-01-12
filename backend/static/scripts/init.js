// @ts-check
'use strict';
export {};

// Get a new version each day, so caching can still be effective
const {version} = await import(`/scripts/version.js?v=${Math.floor(Date.now() / 86400000)}`);

const params = new URL(window.location.href).searchParams;
params.set('v', version);
window.history.replaceState(null, '', `?${params.toString()}`);
const script = document.createElement('script');
script.setAttribute('src', `/dist/index.bundle.js?v=${version}`);
document.head.appendChild(script);
