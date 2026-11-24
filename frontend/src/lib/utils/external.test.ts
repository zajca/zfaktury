import { describe, it, expect, vi, beforeEach } from 'vitest';
import { prepareExternalOpen } from './external';

describe('prepareExternalOpen in browser mode', () => {
	let tab: { opener: unknown; location: { href: string }; close: ReturnType<typeof vi.fn> };

	beforeEach(() => {
		tab = { opener: 'unset', location: { href: '' }, close: vi.fn() };
		vi.stubGlobal(
			'open',
			vi.fn(() => tab)
		);
	});

	it('pre-opens a blank tab synchronously and detaches its opener', () => {
		prepareExternalOpen(false);

		expect(window.open).toHaveBeenCalledWith('', '_blank');
		expect(tab.opener).toBeNull();
	});

	it('points the tab at the URL when opened', async () => {
		const pending = prepareExternalOpen(false);

		await expect(pending.open('https://mojedane.gov.cz/dpr/x')).resolves.toBe(true);
		expect(tab.location.href).toBe('https://mojedane.gov.cz/dpr/x');
	});

	it('closes the pre-opened tab on cancel', () => {
		prepareExternalOpen(false).cancel();

		expect(tab.close).toHaveBeenCalled();
	});

	it('reports a blocked popup', async () => {
		vi.stubGlobal(
			'open',
			vi.fn(() => null)
		);
		const pending = prepareExternalOpen(false);

		await expect(pending.open('https://mojedane.gov.cz/dpr/x')).resolves.toBe(false);
		expect(() => pending.cancel()).not.toThrow();
	});
});

describe('prepareExternalOpen in desktop mode', () => {
	it('does not open a browser tab', () => {
		const openSpy = vi.fn();
		vi.stubGlobal('open', openSpy);

		prepareExternalOpen(true);

		expect(openSpy).not.toHaveBeenCalled();
	});
});
