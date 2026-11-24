import { EPO_ALLOWED_HOST, EPO_FORM_PATH_PREFIX } from '$lib/config/epo';

/**
 * Reports whether a link returned by the backend is an EPO form link: https,
 * the EPO host, a path under the form prefix and no credentials.
 */
export function isEPOFormUrl(raw: string): boolean {
	let url: URL;
	try {
		url = new URL(raw);
	} catch {
		return false;
	}
	return (
		url.protocol === 'https:' &&
		url.hostname === EPO_ALLOWED_HOST &&
		url.port === '' &&
		url.pathname.startsWith(EPO_FORM_PATH_PREFIX) &&
		url.username === '' &&
		url.password === ''
	);
}
