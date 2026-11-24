import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { downloadFile, parseContentDispositionFilename } from './download';

describe('parseContentDispositionFilename', () => {
	it.each([
		['attachment; filename=dph-2025-03.xml', 'dph-2025-03.xml'],
		['attachment; filename="kh-2025-q1.xml"', 'kh-2025-q1.xml'],
		['attachment; filename="a \\"b\\".xml"', 'a "b".xml'],
		["attachment; filename*=UTF-8''dpfo%202025.xml", 'dpfo 2025.xml'],
		["attachment; filename=fallback.xml; filename*=UTF-8''pref.xml", 'pref.xml'],
		["attachment; filename*=UTF-8''%E0%A4%A; filename=plain.xml", 'plain.xml'],
		['ATTACHMENT; FILENAME=upper.xml', 'upper.xml']
	])('parses %s', (header, expected) => {
		expect(parseContentDispositionFilename(header)).toBe(expected);
	});

	it.each([[null], [''], ['attachment'], ['attachment; filename=']])(
		'returns null for %s',
		(header) => {
			expect(parseContentDispositionFilename(header)).toBeNull();
		}
	);
});

describe('downloadFile', () => {
	const mockFetch = vi.fn();
	let anchor: HTMLAnchorElement;

	beforeEach(() => {
		mockFetch.mockReset();
		vi.stubGlobal('fetch', mockFetch);
		URL.createObjectURL = vi.fn(() => 'blob:test');
		URL.revokeObjectURL = vi.fn();
		const create = document.createElement.bind(document);
		vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
			const el = create(tag);
			if (tag === 'a') {
				anchor = el as HTMLAnchorElement;
				vi.spyOn(anchor, 'click').mockImplementation(() => {});
			}
			return el;
		});
	});

	afterEach(() => {
		vi.restoreAllMocks();
		vi.unstubAllGlobals();
	});

	it('uses the name from Content-Disposition', async () => {
		mockFetch.mockResolvedValue(
			new Response('x', {
				status: 200,
				headers: { 'Content-Disposition': 'attachment; filename=dph-2025-q1.xml' }
			})
		);
		await downloadFile('/api/v1/vat-returns/1/xml', 'fallback.xml');
		expect(anchor.download).toBe('dph-2025-q1.xml');
	});

	it('falls back to the argument when the header is missing', async () => {
		mockFetch.mockResolvedValue(new Response('x', { status: 200 }));
		await downloadFile('/api/v1/x', 'fallback.xml');
		expect(anchor.download).toBe('fallback.xml');
	});

	it('throws on a failed response', async () => {
		mockFetch.mockResolvedValue(new Response('no', { status: 500, statusText: 'Error' }));
		await expect(downloadFile('/api/v1/x')).rejects.toThrow('Download failed');
	});
});
