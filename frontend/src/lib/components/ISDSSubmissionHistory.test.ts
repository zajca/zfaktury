import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup, within } from '@testing-library/svelte';
import ISDSSubmissionHistory from './ISDSSubmissionHistory.svelte';
import type { FilingSubmission, FilingSubmissionStatus } from '$lib/api/client';
import { jsonResponse, makeSubmission } from '$lib/utils/isds-test-fetch';

vi.mock('$app/environment', () => ({ browser: true }));
vi.mock('$lib/utils/download', async (orig) => ({
	...(await orig<typeof import('$lib/utils/download')>()),
	downloadFile: vi.fn(async () => {})
}));

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

let rows: FilingSubmission[] = [];
let refreshResponse: () => Response = () => jsonResponse({});
let resolveResponse: () => Response = () => jsonResponse({});

function callsTo(suffix: string) {
	return mockFetch.mock.calls.filter(([url]) => String(url).endsWith(suffix));
}

function bodyOf(call: unknown[]): Record<string, unknown> {
	return JSON.parse((call[1] as RequestInit).body as string);
}

beforeEach(() => {
	mockFetch.mockReset();
	rows = [];
	mockFetch.mockImplementation(async (url: string) => {
		if (url.endsWith('/isds/submissions')) return jsonResponse(rows);
		if (url.endsWith('/refresh')) return refreshResponse();
		if (url.endsWith('/resolve')) return resolveResponse();
		throw new Error(`unexpected call ${url}`);
	});
});

afterEach(() => {
	cleanup();
});

async function renderHistory() {
	const ondocumentfiled = vi.fn();
	render(ISDSSubmissionHistory, { kind: 'vat-return', id: 1, ondocumentfiled });
	await waitFor(() => expect(screen.queryByRole('status')).not.toBeInTheDocument());
	return { ondocumentfiled };
}

describe('ISDSSubmissionHistory', () => {
	it('shows an empty state when nothing was sent', async () => {
		await renderHistory();
		expect(screen.getByText(/zatím nebylo odesláno datovou schránkou/)).toBeInTheDocument();
	});

	it('shows a loading status while the list loads', async () => {
		render(ISDSSubmissionHistory, { kind: 'vat-return', id: 1 });
		const status = screen.getByRole('status');
		expect(status).toHaveTextContent('Nacitani...');
		await waitFor(() => expect(screen.queryByRole('status')).not.toBeInTheDocument());
	});

	it('loads the list on mount and never refreshes by itself', async () => {
		rows = [makeSubmission({ id: 1, status: 'sent' })];
		await renderHistory();

		expect(callsTo('/isds/submissions')).toHaveLength(1);
		expect(callsTo('/refresh')).toHaveLength(0);
		expect(callsTo('/receipt.zfo')).toHaveLength(0);
	});

	it('shows a Czech alert when the list cannot be loaded', async () => {
		mockFetch.mockImplementation(async () =>
			jsonResponse({ error: 'Podání nebylo nalezeno.' }, 404)
		);
		await renderHistory();
		expect(screen.getByRole('alert')).toHaveTextContent('Podání nebylo nalezeno.');
	});

	it('renders one row per status with the Czech label, environment, recipient and ids', async () => {
		const statuses: [FilingSubmissionStatus, string][] = [
			['sending', 'Odesílá se'],
			['unknown', 'Výsledek není znám'],
			['sent', 'Odesláno'],
			['delivered', 'Doručeno'],
			['undeliverable', 'Nedoručitelné'],
			['failed', 'Neodesláno']
		];
		rows = statuses.map(([status], i) =>
			makeSubmission({
				id: i + 1,
				status,
				environment: status === 'sent' ? 'test' : 'production',
				dmId: status === 'sending' || status === 'unknown' ? '' : `10${i}`,
				deliveredAt: status === 'delivered' ? '2026-03-11T08:30:00Z' : null,
				hasReceipt: status === 'delivered'
			})
		);
		await renderHistory();

		for (const [i, [status, label]] of statuses.entries()) {
			const row = screen.getByTestId(`isds-submission-${i + 1}`);
			expect(row, status).toHaveTextContent(label);
			expect(row).toHaveTextContent('Finanční úřad pro hlavní město Prahu');
			expect(row).toHaveTextContent('(7nyn2d9)');
		}
		expect(screen.getByTestId('isds-submission-3')).toHaveTextContent('Testovací prostředí');
		expect(screen.getByTestId('isds-submission-1')).toHaveTextContent('Produkce');
		expect(screen.getByTestId('isds-submission-3')).toHaveTextContent('102');
		expect(screen.getByTestId('isds-submission-4')).toHaveTextContent('11. 3. 2026');
	});

	it('refreshes only on click, updates the row and does not reload the page for a sent row', async () => {
		rows = [makeSubmission({ id: 5, status: 'sent' })];
		refreshResponse = () =>
			jsonResponse({
				submission: makeSubmission({ id: 5, status: 'sent', dmMessageStatus: 3 }),
				documentFiled: false
			});
		const { ondocumentfiled } = await renderHistory();

		await fireEvent.click(screen.getByRole('button', { name: 'Zjistit stav' }));

		await waitFor(() => expect(callsTo('/submissions/5/refresh')).toHaveLength(1));
		await waitFor(() => expect(screen.getByRole('button', { name: 'Zjistit stav' })).toBeEnabled());
		expect(ondocumentfiled).not.toHaveBeenCalled();
	});

	it('shows the delivered state and reloads the document when the refresh files it', async () => {
		rows = [makeSubmission({ id: 5, status: 'sent' })];
		refreshResponse = () =>
			jsonResponse({
				submission: makeSubmission({
					id: 5,
					status: 'delivered',
					deliveredAt: '2026-03-11T08:30:00Z',
					hasReceipt: true
				}),
				documentFiled: true
			});
		const { ondocumentfiled } = await renderHistory();

		await fireEvent.click(screen.getByRole('button', { name: 'Zjistit stav' }));

		await waitFor(() => expect(ondocumentfiled).toHaveBeenCalledTimes(1));
		const row = screen.getByTestId('isds-submission-5');
		expect(row).toHaveTextContent('Doručeno');
		expect(within(row).getByRole('button', { name: 'Stáhnout doručenku' })).toBeInTheDocument();
	});

	it('shows the server message when the refresh fails', async () => {
		rows = [makeSubmission({ id: 5, status: 'sent' })];
		refreshResponse = () =>
			jsonResponse(
				{ error: 'Datová schránka neodpověděla včas. Stav doručení se nepodařilo zjistit.' },
				504
			);
		await renderHistory();

		await fireEvent.click(screen.getByRole('button', { name: 'Zjistit stav' }));

		const alert = await within(screen.getByTestId('isds-submission-5')).findByRole('alert');
		expect(alert).toHaveTextContent('Datová schránka neodpověděla včas.');
	});

	it('offers the receipt only for a delivered row that has one and downloads it', async () => {
		rows = [
			makeSubmission({ id: 1, status: 'sent' }),
			makeSubmission({ id: 2, status: 'delivered', hasReceipt: false }),
			makeSubmission({ id: 3, status: 'delivered', hasReceipt: true, dmId: '555' })
		];
		await renderHistory();

		expect(screen.getAllByRole('button', { name: 'Stáhnout doručenku' })).toHaveLength(1);
		await fireEvent.click(screen.getByRole('button', { name: 'Stáhnout doručenku' }));

		const { downloadFile } = await import('$lib/utils/download');
		await waitFor(() =>
			expect(downloadFile).toHaveBeenCalledWith(
				'/api/v1/filings/submissions/3/receipt.zfo',
				'dorucenka-555.zfo'
			)
		);
	});

	it('shows a Czech message when the receipt download fails', async () => {
		rows = [makeSubmission({ id: 3, status: 'delivered', hasReceipt: true })];
		const { downloadFile } = await import('$lib/utils/download');
		(downloadFile as ReturnType<typeof vi.fn>).mockRejectedValueOnce(
			new Error('Download failed: 404 Not Found')
		);
		await renderHistory();

		await fireEvent.click(screen.getByRole('button', { name: 'Stáhnout doručenku' }));

		const alert = await screen.findByRole('alert');
		expect(alert).toHaveTextContent('Doručenku se nepodařilo stáhnout.');
		expect(alert).not.toHaveTextContent('404');
	});

	describe('rows with an unknown outcome', () => {
		it('explains the situation and offers both resolutions instead of a refresh', async () => {
			rows = [
				makeSubmission({ id: 1, status: 'unknown', dmId: '' }),
				makeSubmission({ id: 2, status: 'sending', dmId: '' })
			];
			await renderHistory();

			for (const id of [1, 2]) {
				const row = screen.getByTestId(`isds-submission-${id}`);
				expect(row).toHaveTextContent('zpráva mohla odejít, ale nemusela');
				expect(row).toHaveTextContent('odeslané zprávy v datové schránce');
				expect(within(row).getByRole('button', { name: 'Zprávu jsem našel' })).toBeInTheDocument();
				expect(within(row).getByRole('button', { name: 'Nic neodešlo' })).toBeInTheDocument();
				expect(within(row).queryByRole('button', { name: 'Zjistit stav' })).not.toBeInTheDocument();
			}
		});

		it('attaches a message id typed as digits only', async () => {
			rows = [makeSubmission({ id: 1, status: 'unknown', dmId: '' })];
			resolveResponse = () =>
				jsonResponse(makeSubmission({ id: 1, status: 'sent', dmId: '987654321' }));
			await renderHistory();

			await fireEvent.click(screen.getByRole('button', { name: 'Zprávu jsem našel' }));
			const input = screen.getByLabelText('Číslo nalezené zprávy') as HTMLInputElement;
			await fireEvent.input(input, { target: { value: '98a76-54 321' } });
			expect(input.value).toBe('987654321');

			await fireEvent.click(screen.getByRole('button', { name: 'Uložit číslo zprávy' }));

			await waitFor(() => expect(callsTo('/resolve')).toHaveLength(1));
			expect(bodyOf(callsTo('/resolve')[0])).toEqual({ action: 'attach', dmId: '987654321' });
			await waitFor(() =>
				expect(screen.getByTestId('isds-submission-1')).toHaveTextContent('Odesláno')
			);
			expect(screen.getByTestId('isds-submission-1')).toHaveTextContent('987654321');
			expect(screen.queryByLabelText('Číslo nalezené zprávy')).not.toBeInTheDocument();
		});

		it('keeps the save button disabled while the id is empty', async () => {
			rows = [makeSubmission({ id: 1, status: 'unknown', dmId: '' })];
			await renderHistory();

			await fireEvent.click(screen.getByRole('button', { name: 'Zprávu jsem našel' }));
			expect(screen.getByRole('button', { name: 'Uložit číslo zprávy' })).toBeDisabled();
			await fireEvent.input(screen.getByLabelText('Číslo nalezené zprávy'), {
				target: { value: 'abc' }
			});
			expect(screen.getByRole('button', { name: 'Uložit číslo zprávy' })).toBeDisabled();
			expect(callsTo('/resolve')).toHaveLength(0);
		});

		it('shows the server refusal when the message id is already used', async () => {
			rows = [makeSubmission({ id: 1, status: 'unknown', dmId: '' })];
			resolveResponse = () =>
				jsonResponse(
					{
						error:
							'Toto číslo zprávy už patří k jinému odeslání. Zkontrolujte ho v datové schránce.'
					},
					409
				);
			await renderHistory();

			await fireEvent.click(screen.getByRole('button', { name: 'Zprávu jsem našel' }));
			await fireEvent.input(screen.getByLabelText('Číslo nalezené zprávy'), {
				target: { value: '42' }
			});
			await fireEvent.click(screen.getByRole('button', { name: 'Uložit číslo zprávy' }));

			const alert = await within(screen.getByTestId('isds-submission-1')).findByRole('alert');
			expect(alert).toHaveTextContent('Toto číslo zprávy už patří k jinému odeslání.');
			expect(screen.getByTestId('isds-submission-1')).toHaveTextContent('Výsledek není znám');
		});

		it('declares nothing sent only after the confirmation', async () => {
			rows = [makeSubmission({ id: 1, status: 'unknown', dmId: '' })];
			resolveResponse = () => jsonResponse(makeSubmission({ id: 1, status: 'failed', dmId: '' }));
			await renderHistory();

			await fireEvent.click(screen.getByRole('button', { name: 'Nic neodešlo' }));
			const dialog = await screen.findByRole('alertdialog');
			expect(callsTo('/resolve')).toHaveLength(0);

			await fireEvent.click(within(dialog).getByRole('button', { name: 'Zrušit' }));
			await waitFor(() => expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument());
			expect(callsTo('/resolve')).toHaveLength(0);

			await fireEvent.click(screen.getByRole('button', { name: 'Nic neodešlo' }));
			const again = await screen.findByRole('alertdialog');
			await fireEvent.click(within(again).getByRole('button', { name: 'Nic neodešlo' }));

			await waitFor(() => expect(callsTo('/resolve')).toHaveLength(1));
			expect(bodyOf(callsTo('/resolve')[0])).toEqual({ action: 'notSent' });
			await waitFor(() =>
				expect(screen.getByTestId('isds-submission-1')).toHaveTextContent('Neodesláno')
			);
		});

		it('shows the refusal for a submission that no longer awaits a decision', async () => {
			rows = [makeSubmission({ id: 1, status: 'sending', dmId: '' })];
			resolveResponse = () =>
				jsonResponse({ error: 'Toto odeslání nečeká na potvrzení výsledku.' }, 409);
			await renderHistory();

			await fireEvent.click(screen.getByRole('button', { name: 'Nic neodešlo' }));
			const dialog = await screen.findByRole('alertdialog');
			await fireEvent.click(within(dialog).getByRole('button', { name: 'Nic neodešlo' }));

			const alert = await within(screen.getByTestId('isds-submission-1')).findByRole('alert');
			expect(alert).toHaveTextContent('Toto odeslání nečeká na potvrzení výsledku.');
		});
	});

	it('reloads the list when the page asks for it', async () => {
		const { component } = (() => {
			const result = render(ISDSSubmissionHistory, { kind: 'vat-return', id: 1 });
			return result;
		})();
		await waitFor(() => expect(screen.queryByRole('status')).not.toBeInTheDocument());
		expect(callsTo('/isds/submissions')).toHaveLength(1);

		rows = [makeSubmission({ id: 9, status: 'sent' })];
		await (component as unknown as { reload: () => Promise<void> }).reload();

		await waitFor(() => expect(screen.getByTestId('isds-submission-9')).toBeInTheDocument());
		expect(callsTo('/isds/submissions')).toHaveLength(2);
	});
});
