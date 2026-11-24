import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup, within } from '@testing-library/svelte';
import ISDSSendButton from './ISDSSendButton.svelte';
import type { ISDSCredentialsOverview } from '$lib/api/client';
import {
	bothCredentials,
	jsonResponse,
	makePreview,
	makeSubmission,
	noCredentials,
	productionCredentials,
	testCredentials
} from '$lib/test-utils/isds-test-fetch';

vi.mock('$app/environment', () => ({ browser: true }));

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

let credentials: ISDSCredentialsOverview = productionCredentials;
let previewResponse: () => Response | Promise<Response>;
let sendResponse: () => Response | Promise<Response>;

function callsTo(suffix: string) {
	return mockFetch.mock.calls.filter(([url]) => String(url).endsWith(suffix));
}

function bodyOf(call: unknown[]): Record<string, unknown> {
	return JSON.parse((call[1] as RequestInit).body as string);
}

beforeEach(() => {
	mockFetch.mockReset();
	credentials = productionCredentials;
	previewResponse = () => jsonResponse(makePreview());
	sendResponse = () => jsonResponse(makeSubmission({ status: 'sent' }), 201);
	mockFetch.mockImplementation(async (url: string) => {
		if (url.endsWith('/settings/isds-credentials')) return jsonResponse(credentials);
		if (url.endsWith('/isds/preview')) return previewResponse();
		if (url.endsWith('/isds/send')) return sendResponse();
		throw new Error(`unexpected call ${url}`);
	});
});

afterEach(() => {
	cleanup();
});

const sendButtonName = 'Odeslat datovou schránkou';

async function renderButton(props: Partial<{ filed: boolean; hasXml: boolean }> = {}) {
	const onsent = vi.fn();
	const onattempt = vi.fn();
	render(ISDSSendButton, {
		kind: 'vat-return',
		id: 7,
		hasXml: props.hasXml ?? true,
		filed: props.filed ?? false,
		onsent,
		onattempt
	});
	await waitFor(() => expect(callsTo('/settings/isds-credentials')).toHaveLength(1));
	return { onsent, onattempt };
}

async function openDialog() {
	const button = await screen.findByRole('button', { name: sendButtonName });
	await waitFor(() => expect(button).toBeEnabled());
	await fireEvent.click(button);
	return screen.findByRole('dialog');
}

describe('ISDSSendButton', () => {
	it('is disabled with a hint and a settings link when no login is saved', async () => {
		credentials = noCredentials;
		await renderButton();

		const hint = await screen.findByTestId('isds-no-login');
		expect(hint).toHaveTextContent('nejdřív uložte přihlašovací údaje');
		expect(within(hint).getByRole('link', { name: 'nastavení' })).toHaveAttribute(
			'href',
			'/settings/firma'
		);
		expect(screen.getByRole('button', { name: sendButtonName })).toBeDisabled();
		expect(callsTo('/isds/preview')).toHaveLength(0);
	});

	it('is disabled with a reason when the document is already filed', async () => {
		await renderButton({ filed: true });

		await waitFor(() => expect(screen.getByTestId('isds-env-single')).toBeInTheDocument());
		const button = screen.getByRole('button', { name: sendButtonName });
		expect(button).toBeDisabled();
		expect(button).toHaveAttribute('title', 'Podání je už označeno jako podané.');
		expect(screen.getByText(/další odeslání není možné/)).toBeInTheDocument();
	});

	it('is disabled until XML is generated', async () => {
		await renderButton({ hasXml: false });

		await waitFor(() => expect(screen.getByTestId('isds-env-single')).toBeInTheDocument());
		const button = screen.getByRole('button', { name: sendButtonName });
		expect(button).toBeDisabled();
		expect(button).toHaveAttribute('title', 'Nejdřív vygenerujte XML.');
	});

	it('uses the only saved environment and shows it', async () => {
		await renderButton();

		const single = await screen.findByTestId('isds-env-single');
		expect(single).toHaveTextContent('Produkce');
		expect(screen.queryByRole('radio')).not.toBeInTheDocument();

		await openDialog();
		expect(callsTo('/isds/preview')).toHaveLength(1);
		expect(bodyOf(callsTo('/isds/preview')[0])).toEqual({ environment: 'production' });
	});

	it('shows the test environment name for a test-only login', async () => {
		credentials = testCredentials;
		await renderButton();

		expect(await screen.findByTestId('isds-env-single')).toHaveTextContent('Testovací prostředí');
	});

	it('makes the user pick when both environments are saved', async () => {
		credentials = bothCredentials;
		await renderButton();

		const production = await screen.findByRole('radio', { name: 'Produkce' });
		const test = screen.getByRole('radio', { name: 'Testovací prostředí' });
		expect(production).not.toBeChecked();
		expect(test).not.toBeChecked();
		const button = screen.getByRole('button', { name: sendButtonName });
		expect(button).toBeDisabled();
		expect(button).toHaveAttribute('title', 'Vyberte prostředí.');

		await fireEvent.click(test);
		await waitFor(() => expect(button).toBeEnabled());
		previewResponse = () => jsonResponse(makePreview({ environment: 'test' }));
		await fireEvent.click(button);

		const dialog = await screen.findByRole('dialog');
		expect(bodyOf(callsTo('/isds/preview')[0])).toEqual({ environment: 'test' });
		expect(within(dialog).getByText('Testovací prostředí')).toBeInTheDocument();
	});

	it('shows the recipient, environment, file and findings in the confirmation dialog', async () => {
		previewResponse = () =>
			jsonResponse(
				makePreview({
					findings: [
						{
							type: 'P',
							code: 'X1',
							field: 'd_poddp',
							section: 'VetaD',
							row: '12',
							text: 'Ověřte termín podání.'
						}
					]
				})
			);
		await renderButton();
		const dialog = await openDialog();

		expect(dialog).toHaveTextContent('Produkce');
		expect(dialog).toHaveTextContent('Finanční úřad pro hlavní město Prahu');
		expect(dialog).toHaveTextContent('(7nyn2d9)');
		expect(dialog).toHaveTextContent('dph-2026-03.xml');
		expect(dialog).toHaveTextContent('1,5 kB');
		expect(dialog).toHaveTextContent('Daňové přiznání k DPH za 3/2026');
		const findings = within(dialog).getByTestId('isds-findings');
		expect(findings).toHaveTextContent('Varování: Ověřte termín podání.');
		expect(findings).toHaveTextContent('VetaD, d_poddp, řádek 12');
	});

	it('sends without a tick when the findings do not require acceptance', async () => {
		const { onsent } = await renderButton();
		const dialog = await openDialog();

		expect(within(dialog).queryByRole('checkbox')).not.toBeInTheDocument();
		await fireEvent.click(within(dialog).getByRole('button', { name: 'Odeslat' }));

		await waitFor(() => expect(onsent).toHaveBeenCalledTimes(1));
		expect(bodyOf(callsTo('/isds/send')[0])).toEqual({
			environment: 'production',
			confirmToken: 'token-1',
			acceptWarnings: false
		});
		expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
	});

	it('keeps the send button disabled until "Odeslat i přesto" is ticked', async () => {
		previewResponse = () =>
			jsonResponse(
				makePreview({
					requiresAcceptWarnings: true,
					findings: [{ type: 'P', code: '', field: '', section: '', row: '', text: 'Po lhůtě?' }]
				})
			);
		await renderButton();
		const dialog = await openDialog();

		const send = within(dialog).getByRole('button', { name: 'Odeslat' });
		expect(send).toBeDisabled();
		await fireEvent.click(within(dialog).getByRole('checkbox', { name: 'Odeslat i přesto' }));
		await waitFor(() => expect(send).toBeEnabled());
		await fireEvent.click(send);

		await waitFor(() => expect(callsTo('/isds/send')).toHaveLength(1));
		expect(bodyOf(callsTo('/isds/send')[0]).acceptWarnings).toBe(true);
	});

	it('sends once when the send button is clicked twice', async () => {
		let release: (r: Response) => void = () => {};
		sendResponse = () =>
			new Promise<Response>((resolve) => {
				release = resolve;
			});
		const { onsent } = await renderButton();
		const dialog = await openDialog();

		const send = within(dialog).getByRole('button', { name: 'Odeslat' });
		await fireEvent.click(send);
		await fireEvent.click(send);

		await waitFor(() => expect(callsTo('/isds/send')).toHaveLength(1));
		expect(within(dialog).getByRole('button', { name: 'Odesílám...' })).toBeDisabled();
		release(jsonResponse(makeSubmission(), 201));
		await waitFor(() => expect(onsent).toHaveBeenCalledTimes(1));
		expect(callsTo('/isds/send')).toHaveLength(1);
	});

	it('shows the server message on an expired token and offers a new preview without re-sending', async () => {
		sendResponse = () =>
			jsonResponse({ error: 'Potvrzení vypršelo, bylo už použito. Zobrazte náhled znovu.' }, 409);
		const { onattempt } = await renderButton();
		const dialog = await openDialog();

		await fireEvent.click(within(dialog).getByRole('button', { name: 'Odeslat' }));

		const alert = await within(dialog).findByRole('alert');
		expect(alert).toHaveTextContent('Potvrzení vypršelo, bylo už použito. Zobrazte náhled znovu.');
		expect(onattempt).toHaveBeenCalledTimes(1);
		expect(within(dialog).queryByRole('button', { name: 'Odeslat' })).not.toBeInTheDocument();
		expect(callsTo('/isds/send')).toHaveLength(1);

		previewResponse = () => jsonResponse(makePreview({ confirmToken: 'token-2' }));
		await fireEvent.click(within(dialog).getByRole('button', { name: 'Zobrazit náhled znovu' }));
		await waitFor(() => expect(callsTo('/isds/preview')).toHaveLength(2));
		await waitFor(() =>
			expect(within(dialog).getByRole('button', { name: 'Odeslat' })).toBeEnabled()
		);
		expect(within(dialog).queryByRole('alert')).not.toBeInTheDocument();
		// A new preview never sends by itself.
		expect(callsTo('/isds/send')).toHaveLength(1);
	});

	it('shows a Czech message instead of raw text when the server is unreachable', async () => {
		sendResponse = () => {
			throw new TypeError('Failed to fetch');
		};
		await renderButton();
		const dialog = await openDialog();

		await fireEvent.click(within(dialog).getByRole('button', { name: 'Odeslat' }));

		const alert = await within(dialog).findByRole('alert');
		expect(alert).toHaveTextContent('Nepodařilo se spojit se serverem.');
		expect(alert).not.toHaveTextContent('Failed to fetch');
	});

	it('shows the server message and findings when the preview is rejected', async () => {
		previewResponse = () =>
			jsonResponse(
				{
					error: 'EPO v podání našlo chyby, nic nebylo odesláno.',
					findings: [
						{ type: 'K', code: 'K1', field: 'c_ufo', section: 'VetaD', row: '', text: 'Chybí FÚ.' }
					]
				},
				422
			);
		await renderButton();
		const button = await screen.findByRole('button', { name: sendButtonName });
		await waitFor(() => expect(button).toBeEnabled());
		await fireEvent.click(button);

		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent('EPO v podání našlo chyby, nic nebylo odesláno.');
		expect(alert).toHaveTextContent('Kritická chyba: Chybí FÚ.');
		expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
		expect(callsTo('/isds/send')).toHaveLength(0);
	});

	it('closes the dialog with Escape and Cancel without sending', async () => {
		await renderButton();
		const dialog = await openDialog();

		await fireEvent.keyDown(window, { key: 'Escape' });
		await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());

		const again = await openDialog();
		await fireEvent.click(within(again).getByRole('button', { name: 'Zrušit' }));
		await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
		expect(dialog).not.toBeInTheDocument();
		expect(callsTo('/isds/send')).toHaveLength(0);
	});

	it('shows a Czech alert when the saved logins cannot be read', async () => {
		mockFetch.mockImplementation(async () =>
			jsonResponse({ error: 'Klíčenka není dostupná.' }, 503)
		);
		render(ISDSSendButton, { kind: 'vat-return', id: 7, hasXml: true, filed: false });

		expect(await screen.findByRole('alert')).toHaveTextContent('Klíčenka není dostupná.');
		expect(screen.getByRole('button', { name: sendButtonName })).toBeDisabled();
	});
});
