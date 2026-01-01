// @ts-check
'use strict';
export { };

// Get a new version each hour, so caching can still be effective
const { version } = await import(`/scripts/version.js?v=${Math.floor(Date.now() / (1000 * 60 * 60))}`);
const params = new URL(window.location.href).searchParams;
params.set('v', version);
window.history.replaceState(null, '', `?${params.toString()}`);

/**
 * Helper to inject scripts as Promises to ensure order
 * @param {string} name 
 */
const injectBundle = (name) => {
	return new Promise((resolve, reject) => {
		const script = document.createElement('script');
		script.setAttribute('src', `/dist/${name}.bundle.js?v=${version}`);
		script.onload = resolve;
		script.onerror = reject;
		document.head.appendChild(script);
	});
};

// Access the script element and read the data attribute
const scriptUrl = new URL(import.meta.url);
const page = scriptUrl.searchParams.get('page');

if (!page) {
	console.error(`Trying to init invalid page: ${page}`);
} else {
	if (page !== 'index') {
		// Always inject index bundle first
		await injectBundle('index');
	}
	await injectBundle(page);
}