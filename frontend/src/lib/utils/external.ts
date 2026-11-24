// Opens an external link in the system browser (desktop) or a new tab (browser mode).

const BROWSER_SERVICE_FQN = 'github.com/zajca/zfaktury/internal/desktop.BrowserService';

/** A link opening prepared synchronously in a click handler. */
export interface PendingExternalOpen {
	/** Opens the link; resolves false when the browser blocked the new tab. */
	open(url: string): Promise<boolean>;
	/** Releases whatever was prepared (closes the pre-opened tab). */
	cancel(): void;
}

async function openInDesktop(url: string): Promise<void> {
	// Use a variable so Vite does not analyze the path; Wails injects the runtime.
	const wailsPath = '/wails/runtime.js';
	const mod = await import(/* @vite-ignore */ wailsPath);
	await mod.Call.ByName(`${BROWSER_SERVICE_FQN}.OpenExternal`, url);
}

/**
 * Prepares opening an external link. Call it synchronously in the click handler,
 * before any await: in browser mode it pre-opens a blank tab, which browsers only
 * allow while the user gesture is active, and the tab is pointed at the URL once
 * the API call has returned.
 * @param desktop - result of `detectDesktop()`
 */
export function prepareExternalOpen(desktop: boolean): PendingExternalOpen {
	if (desktop) {
		return {
			async open(url) {
				await openInDesktop(url);
				return true;
			},
			cancel() {}
		};
	}

	const tab = window.open('', '_blank');
	if (tab) {
		tab.opener = null;
	}
	return {
		async open(url) {
			if (!tab) return false;
			tab.location.href = url;
			return true;
		},
		cancel() {
			tab?.close();
		}
	};
}
