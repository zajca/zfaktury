// Desktop-aware file download utility.
// In Wails desktop mode, downloads go through the Go DownloadService (native save dialog).
// In browser mode (serve), downloads use the standard blob + anchor pattern.

let desktopMode: boolean | null = null;
let wailsCall: { ByName: (name: string, ...args: unknown[]) => Promise<unknown> } | null = null;

const DOWNLOAD_SERVICE_FQN = 'github.com/zajca/zfaktury/internal/desktop.DownloadService';

async function detectDesktop(): Promise<boolean> {
	if (desktopMode !== null) return desktopMode;
	try {
		// Use variable to prevent Vite from statically analyzing the import path.
		// /wails/runtime.js is injected by Wails at runtime in the desktop webview.
		const wailsPath = '/wails/runtime.js';
		const mod = await import(/* @vite-ignore */ wailsPath);
		wailsCall = mod.Call;
		desktopMode = true;
	} catch {
		desktopMode = false;
	}
	return desktopMode;
}

/** Eagerly detect desktop mode (call from layout onMount). */
export function isDesktopMode(): Promise<boolean> {
	return detectDesktop();
}

/**
 * Extracts the filename from a Content-Disposition header value.
 * Prefers the RFC 5987 `filename*=` form over `filename=`; returns null when
 * the header carries no usable name.
 */
export function parseContentDispositionFilename(header: string | null): string | null {
	if (!header) return null;

	const extended = /filename\*\s*=\s*([^;]+)/i.exec(header);
	if (extended) {
		const match = /^([^']*)'[^']*'(.*)$/.exec(extended[1].trim());
		if (match) {
			try {
				const decoded = decodeURIComponent(match[2]);
				if (decoded) return decoded;
			} catch {
				// Malformed percent-encoding: fall through to the plain form.
			}
		}
	}

	const plain = /(?:^|[;\s])filename\s*=\s*(?:"((?:[^"\\]|\\.)*)"|([^;\s]+))/i.exec(header);
	if (plain) {
		const value = plain[1] !== undefined ? plain[1].replace(/\\(.)/g, '$1') : plain[2];
		if (value) return value;
	}
	return null;
}

/**
 * Download a file from the API. The file name comes from the response's
 * Content-Disposition header.
 * @param apiPath - API path, e.g. "/api/v1/invoices/5/pdf"
 * @param fallbackFilename - Used only when the response carries no filename
 */
export async function downloadFile(apiPath: string, fallbackFilename?: string): Promise<void> {
	const isDesktop = await detectDesktop();

	if (isDesktop && wailsCall) {
		await wailsCall.ByName(`${DOWNLOAD_SERVICE_FQN}.SaveToFile`, apiPath, fallbackFilename ?? '');
		return;
	}

	// Browser fallback: fetch blob + programmatic anchor click
	const response = await fetch(apiPath);
	if (!response.ok) {
		throw new Error(`Download failed: ${response.status} ${response.statusText}`);
	}
	const blob = await response.blob();
	const url = URL.createObjectURL(blob);
	const a = document.createElement('a');
	a.href = url;
	a.download =
		parseContentDispositionFilename(response.headers.get('Content-Disposition')) ??
		fallbackFilename ??
		'download';
	document.body.appendChild(a);
	a.click();
	document.body.removeChild(a);
	URL.revokeObjectURL(url);
}
