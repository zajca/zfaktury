import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$lib/utils/download', () => ({ detectDesktop: vi.fn(async () => true) }));

const openUrl = vi.fn(async () => true);
const cancel = vi.fn();
vi.mock('$lib/utils/external', () => ({
	prepareExternalOpen: vi.fn(() => ({ open: openUrl, cancel }))
}));

import EPOOpenButton from './EPOOpenButton.svelte';

const FORM_URL = 'https://mojedane.gov.cz/dpr/adis/idpr_epo/epo2/form/form_uvod.faces?form_id=FAKE';

beforeEach(() => {
	mockFetch.mockReset();
	openUrl.mockClear();
	cancel.mockClear();
});

afterEach(() => {
	cleanup();
});

describe('EPOOpenButton in desktop mode', () => {
	it('hands the URL to the desktop opener instead of a browser tab', async () => {
		mockFetch.mockResolvedValueOnce(
			new Response(JSON.stringify({ url: FORM_URL, warnings: [] }), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			})
		);
		render(EPOOpenButton, { props: { kind: 'income-tax-return', id: 3, hasXml: true } });
		const button = screen.getByRole('button', { name: /Otevřít v EPO/ });
		await waitFor(() => expect(button).toBeEnabled());

		await fireEvent.click(button);

		await waitFor(() => expect(openUrl).toHaveBeenCalledWith(FORM_URL));
		const { prepareExternalOpen } = await import('$lib/utils/external');
		expect(prepareExternalOpen).toHaveBeenCalledWith(true);
		expect(await screen.findByText('Formulář byl otevřen v prohlížeči.')).toBeInTheDocument();
	});

	it('shows the error and cancels when the desktop opener refuses the link', async () => {
		mockFetch.mockResolvedValueOnce(
			new Response(JSON.stringify({ url: FORM_URL, warnings: [] }), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			})
		);
		openUrl.mockRejectedValueOnce(new Error('refusing to open link'));
		render(EPOOpenButton, { props: { kind: 'vat-return', id: 3, hasXml: true } });
		const button = screen.getByRole('button', { name: /Otevřít v EPO/ });
		await waitFor(() => expect(button).toBeEnabled());

		await fireEvent.click(button);

		expect(await screen.findByRole('alert')).toHaveTextContent('refusing to open link');
		expect(cancel).toHaveBeenCalled();
	});
});
