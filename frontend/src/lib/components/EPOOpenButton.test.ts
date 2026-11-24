import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$lib/utils/download', () => ({ detectDesktop: vi.fn(async () => false) }));

import EPOOpenButton from './EPOOpenButton.svelte';

const FORM_URL = 'https://mojedane.gov.cz/dpr/adis/idpr_epo/epo2/form/form_uvod.faces?form_id=FAKE';

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const warning = {
	type: 'P',
	code: '',
	field: 'c_okec',
	section: 'VetaD',
	row: '',
	text: 'Hlavní ekonomická činnost by měla odpovídat číselníku.'
};

const critical = {
	type: 'K',
	code: 'POVINNA_POLOZKA',
	field: 'dic',
	section: 'VetaP',
	row: '3',
	text: 'Povinná položka není vyplněna.'
};

interface FakeTab {
	opener: unknown;
	location: { href: string };
	close: ReturnType<typeof vi.fn>;
}

let tab: FakeTab;
let openSpy: ReturnType<typeof vi.fn>;

beforeEach(() => {
	mockFetch.mockReset();
	tab = { opener: 'unset', location: { href: '' }, close: vi.fn() };
	openSpy = vi.fn(() => tab);
	vi.stubGlobal('open', openSpy);
});

afterEach(() => {
	cleanup();
});

async function renderButton(props: Record<string, unknown> = {}) {
	const result = render(EPOOpenButton, {
		props: { kind: 'vat-return', id: 7, hasXml: true, ...props }
	});
	const button = screen.getByRole('button', { name: /Otevřít v EPO/ });
	await waitFor(() => expect(button).toBeEnabled());
	return { ...result, button };
}

describe('EPOOpenButton', () => {
	it('is disabled without XML and shows the hint', () => {
		render(EPOOpenButton, { props: { kind: 'vat-return', id: 7, hasXml: false } });

		expect(screen.getByRole('button', { name: /Otevřít v EPO/ })).toBeDisabled();
		expect(screen.getByText('Nejdřív vygenerujte XML.')).toBeInTheDocument();
	});

	it('shows the note about the browser and the 30 minute link', async () => {
		await renderButton();

		expect(screen.getByText(/Odkaz platí 30 minut/)).toBeInTheDocument();
	});

	it('posts to the EPO endpoint and opens the returned URL in the pre-opened tab', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ url: FORM_URL, warnings: [warning] }));
		const { button } = await renderButton({ kind: 'vies-summary', id: 12 });

		await fireEvent.click(button);

		await waitFor(() => expect(tab.location.href).toBe(FORM_URL));
		const [url, init] = mockFetch.mock.calls[0];
		expect(url).toBe('/api/v1/epo/vies-summary/12/open-form');
		expect(init.method).toBe('POST');
		expect(openSpy).toHaveBeenCalledWith('', '_blank');
		expect(tab.opener).toBeNull();
		expect(screen.getByText('Formulář byl otevřen v prohlížeči.')).toBeInTheDocument();
		expect(screen.getByText(/Hlavní ekonomická činnost/)).toBeInTheDocument();
		expect(screen.getByText(/Varování/)).toBeInTheDocument();
	});

	it('pre-opens the tab before the API call resolves', async () => {
		let resolveFetch: (r: Response) => void = () => {};
		mockFetch.mockReturnValueOnce(new Promise<Response>((r) => (resolveFetch = r)));
		const { button } = await renderButton();

		await fireEvent.click(button);

		expect(openSpy).toHaveBeenCalledTimes(1);
		expect(tab.location.href).toBe('');
		expect(screen.getByRole('status')).toBeInTheDocument();
		expect(screen.getByText('Nacitani...')).toBeInTheDocument();

		resolveFetch(jsonResponse({ url: FORM_URL, warnings: [] }));
		await waitFor(() => expect(tab.location.href).toBe(FORM_URL));
	});

	it('lists blocking findings on 422, opens nothing and closes the tab', async () => {
		mockFetch.mockResolvedValueOnce(
			jsonResponse(
				{ error: 'EPO v podání našlo chyby, formulář nebyl otevřen.', findings: [critical] },
				422
			)
		);
		const { button } = await renderButton();

		await fireEvent.click(button);

		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent('EPO v podání našlo chyby');
		expect(alert).toHaveTextContent('Kritická chyba');
		expect(alert).toHaveTextContent('Povinná položka není vyplněna.');
		expect(alert).toHaveTextContent('řádek 3');
		expect(tab.location.href).toBe('');
		expect(tab.close).toHaveBeenCalled();
		expect(screen.queryByText('Formulář byl otevřen v prohlížeči.')).not.toBeInTheDocument();
		expect(screen.queryByRole('button', { name: 'Označit jako podané' })).not.toBeInTheDocument();
	});

	it.each([
		[409, 'Nejdřív vygenerujte XML.'],
		[502, 'EPO je nedostupné, stáhněte XML a nahrajte ho ručně.'],
		[504, 'EPO neodpovědělo včas. Zkuste to znovu.']
	])('shows the server message for HTTP %i', async (status, message) => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ error: message }, status));
		const { button } = await renderButton();

		await fireEvent.click(button);

		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent(message);
		expect(tab.close).toHaveBeenCalled();
	});

	it('shows an error when the request itself fails', async () => {
		mockFetch.mockRejectedValueOnce(new Error('Failed to fetch'));
		const { button } = await renderButton();

		await fireEvent.click(button);

		expect(await screen.findByRole('alert')).toHaveTextContent('Failed to fetch');
		expect(tab.close).toHaveBeenCalled();
	});

	it('shows a link when the browser blocked the new tab', async () => {
		openSpy.mockReturnValue(null);
		mockFetch.mockResolvedValueOnce(jsonResponse({ url: FORM_URL, warnings: [] }));
		const { button } = await renderButton();

		await fireEvent.click(button);

		const link = await screen.findByRole('link', { name: 'Otevřít formulář v EPO' });
		expect(link).toHaveAttribute('href', FORM_URL);
		expect(link).toHaveAttribute('target', '_blank');
		expect(link).toHaveAttribute('rel', 'noopener noreferrer');
	});

	it('offers marking the filing as filed after success and does not do it automatically', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ url: FORM_URL, warnings: [] }));
		const onMarkFiled = vi.fn();
		const { button } = await renderButton({ onMarkFiled });

		expect(onMarkFiled).not.toHaveBeenCalled();
		await fireEvent.click(button);

		const mark = await screen.findByRole('button', { name: 'Označit jako podané' });
		expect(onMarkFiled).not.toHaveBeenCalled();
		await fireEvent.click(mark);
		expect(onMarkFiled).toHaveBeenCalledTimes(1);
		expect(mockFetch).toHaveBeenCalledTimes(1);
	});

	it('does not offer marking as filed without the callback', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ url: FORM_URL, warnings: [] }));
		const { button } = await renderButton();

		await fireEvent.click(button);

		await screen.findByText('Formulář byl otevřen v prohlížeči.');
		expect(screen.queryByRole('button', { name: 'Označit jako podané' })).not.toBeInTheDocument();
	});
});
