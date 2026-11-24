import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$lib/utils/download', () => ({ detectDesktop: vi.fn(async () => true) }));

const byName = vi.fn(async (..._args: unknown[]) => undefined);
vi.mock('/wails/runtime.js', () => ({ Call: { ByName: (...args: unknown[]) => byName(...args) } }));

import EPOOpenTestHarness from './EPOOpenTestHarness.svelte';

const FORM_URL = 'https://mojedane.gov.cz/dpr/adis/idpr_epo/epo2/form/form_uvod.faces?form_id=FAKE';
const OPEN_EXTERNAL = 'github.com/zajca/zfaktury/internal/desktop.BrowserService.OpenExternal';

function formResponse() {
	return new Response(JSON.stringify({ url: FORM_URL, warnings: [] }), {
		status: 200,
		headers: { 'Content-Type': 'application/json' }
	});
}

let openSpy: ReturnType<typeof vi.fn>;

beforeEach(() => {
	mockFetch.mockReset();
	byName.mockReset();
	byName.mockResolvedValue(undefined);
	openSpy = vi.fn();
	vi.stubGlobal('open', openSpy);
});

afterEach(() => {
	cleanup();
});

async function renderButton() {
	render(EPOOpenTestHarness, { props: { kind: 'income-tax-return', id: 3 } });
	const button = screen.getByRole('button', { name: /Otevřít v EPO/ });
	await waitFor(() => expect(button).toBeEnabled());
	return button;
}

describe('EPOOpenButton in desktop mode', () => {
	it('calls BrowserService.OpenExternal with the URL and opens no browser tab', async () => {
		mockFetch.mockResolvedValueOnce(formResponse());
		const button = await renderButton();

		await fireEvent.click(button);

		await waitFor(() => expect(byName).toHaveBeenCalledWith(OPEN_EXTERNAL, FORM_URL));
		expect(openSpy).not.toHaveBeenCalled();
		expect(await screen.findByText('Formulář byl otevřen v prohlížeči.')).toBeInTheDocument();
	});

	it('shows a Czech error when the desktop opener rejects the link', async () => {
		mockFetch.mockResolvedValueOnce(formResponse());
		byName.mockRejectedValueOnce(new Error('refusing to open link'));
		const button = await renderButton();

		await fireEvent.click(button);

		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent('Formulář se nepodařilo otevřít v prohlížeči.');
		expect(alert).not.toHaveTextContent('refusing to open link');
		expect(screen.queryByText('Formulář byl otevřen v prohlížeči.')).not.toBeInTheDocument();
	});
});
