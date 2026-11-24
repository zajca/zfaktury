import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$lib/utils/download', () => ({ detectDesktop: vi.fn(async () => false) }));

import { EPOOpenState } from './epo-open.svelte';

const FORM_URL = 'https://mojedane.gov.cz/dpr/adis/idpr_epo/epo2/form/form_uvod.faces?form_id=FAKE';

beforeEach(() => {
	mockFetch.mockReset();
	vi.stubGlobal(
		'open',
		vi.fn(() => ({ opener: null, closed: false, location: { href: '' }, close: vi.fn() }))
	);
});

describe('EPOOpenState.open', () => {
	it('ignores a second call while one is running', async () => {
		let resolveFetch: (r: Response) => void = () => {};
		mockFetch.mockReturnValueOnce(new Promise<Response>((r) => (resolveFetch = r)));
		const state = new EPOOpenState();
		await state.init();

		const first = state.open('vat-return', 1);
		await state.open('vat-return', 1);
		resolveFetch(
			new Response(JSON.stringify({ url: FORM_URL, warnings: [] }), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			})
		);
		await first;

		expect(mockFetch).toHaveBeenCalledTimes(1);
		expect(window.open).toHaveBeenCalledTimes(1);
		expect(state.opened).toBe(true);
	});
});
