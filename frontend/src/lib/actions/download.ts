import { downloadFile, isDesktopMode } from '$lib/utils/download';

/**
 * Svelte action for <a href="..." download> links.
 * In desktop mode, intercepts clicks and routes through the Go DownloadService.
 * In browser mode, does nothing (native <a download> works).
 */
export function nativeDownload(node: HTMLAnchorElement, filename?: string) {
	let isDesktop = false;
	isDesktopMode().then((v) => (isDesktop = v));

	function handleClick(e: Event) {
		if (!isDesktop) return;
		e.preventDefault();
		const href = node.getAttribute('href');
		if (!href) return;
		const name = filename || href.split('/').pop() || 'download';
		downloadFile(href, name);
	}

	node.addEventListener('click', handleClick);

	return {
		update(newFilename?: string) {
			filename = newFilename;
		},
		destroy() {
			node.removeEventListener('click', handleClick);
		}
	};
}
