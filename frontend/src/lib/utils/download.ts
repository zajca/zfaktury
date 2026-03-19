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
 * Download a file from the API.
 * @param apiPath - API path, e.g. "/api/v1/invoices/5/pdf"
 * @param filename - Suggested filename for save dialog / download
 */
export async function downloadFile(apiPath: string, filename: string): Promise<void> {
	const isDesktop = await detectDesktop();

	if (isDesktop && wailsCall) {
		await wailsCall.ByName(`${DOWNLOAD_SERVICE_FQN}.SaveToFile`, apiPath, filename);
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
	a.download = filename;
	document.body.appendChild(a);
	a.click();
	document.body.removeChild(a);
	URL.revokeObjectURL(url);
}
