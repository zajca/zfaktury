import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup, within } from '@testing-library/svelte';
import ISDSCredentialsSection from './ISDSCredentialsSection.svelte';

vi.mock('$app/environment', () => ({ browser: true }));

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const emptyOverview = {
	test: { configured: false, username: '' },
	production: { configured: false, username: '' }
};

const savedTestOverview = {
	test: { configured: true, username: 'sender-user' },
	production: { configured: false, username: '' }
};

function callsWith(method: string) {
	return mockFetch.mock.calls.filter(([, init]) => (init as RequestInit)?.method === method);
}

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('ISDSCredentialsSection', () => {
	it('loads the status on mount and shows both environments', async () => {
		mockFetch.mockResolvedValue(jsonResponse(savedTestOverview));
		render(ISDSCredentialsSection);

		await waitFor(() => {
			expect(screen.getByTestId('isds-status-test')).toHaveTextContent('Přihlášení uloženo');
		});
		expect(screen.getByTestId('isds-status-test')).toHaveTextContent('sender-user');
		expect(screen.getByTestId('isds-status-production')).toHaveTextContent(
			'Přihlášení není uloženo'
		);
		expect(mockFetch).toHaveBeenCalledWith(
			expect.stringContaining('/api/v1/settings/isds-credentials'),
			expect.objectContaining({ method: 'GET' })
		);
	});

	it('shows the delegated-user and keyring note', async () => {
		mockFetch.mockResolvedValue(jsonResponse(emptyOverview));
		render(ISDSCredentialsSection);
		await waitFor(() => expect(screen.getByTestId('isds-status-test')).toBeInTheDocument());

		expect(screen.getByText(/vytvářet a odesílat zprávy/)).toBeInTheDocument();
		expect(screen.getByText(/seznam odeslaných zpráv a doručenky/)).toBeInTheDocument();
		expect(screen.getByText(/doručení přihlášením/)).toBeInTheDocument();
		expect(screen.getByText(/systémové klíčenky a už se nikdy nezobrazí/)).toBeInTheDocument();
	});

	it('uses password inputs with autocomplete off', async () => {
		mockFetch.mockResolvedValue(jsonResponse(emptyOverview));
		render(ISDSCredentialsSection);
		await waitFor(() => expect(screen.getByTestId('isds-status-test')).toBeInTheDocument());

		for (const env of ['test', 'production']) {
			const password = document.getElementById(`isds-password-${env}`) as HTMLInputElement;
			expect(password.type).toBe('password');
			expect(password.getAttribute('autocomplete')).toBe('off');
			const username = document.getElementById(`isds-username-${env}`) as HTMLInputElement;
			expect(username.getAttribute('autocomplete')).toBe('off');
		}
	});

	it('saves with the confirm header, shows the new status and clears the password', async () => {
		mockFetch
			.mockResolvedValueOnce(jsonResponse(emptyOverview))
			.mockResolvedValueOnce(jsonResponse({ configured: true, username: 'sender-user' }));
		render(ISDSCredentialsSection);
		await waitFor(() => expect(screen.getByTestId('isds-status-test')).toBeInTheDocument());

		const username = document.getElementById('isds-username-test') as HTMLInputElement;
		const password = document.getElementById('isds-password-test') as HTMLInputElement;
		await fireEvent.input(username, { target: { value: 'sender-user' } });
		await fireEvent.input(password, { target: { value: 'Sample-Pass-1' } });
		await fireEvent.click(screen.getAllByRole('button', { name: 'Uložit přihlášení' })[0]);

		await waitFor(() => {
			expect(screen.getByTestId('isds-status-test')).toHaveTextContent('Přihlášení uloženo');
		});

		const [url, init] = callsWith('PUT')[0] as [string, RequestInit];
		expect(url).toContain('/api/v1/settings/isds-credentials');
		const headers = init.headers as Record<string, string>;
		expect(headers['X-ZFaktury-Confirm']).toBe('1');
		expect(headers['Content-Type']).toBe('application/json');
		expect(JSON.parse(init.body as string)).toEqual({
			environment: 'test',
			username: 'sender-user',
			password: 'Sample-Pass-1'
		});
		expect(password.value).toBe('');
	});

	it('disables save until username and password are filled', async () => {
		mockFetch.mockResolvedValue(jsonResponse(emptyOverview));
		render(ISDSCredentialsSection);
		await waitFor(() => expect(screen.getByTestId('isds-status-test')).toBeInTheDocument());

		const save = screen.getAllByRole('button', { name: 'Uložit přihlášení' })[0];
		expect(save).toBeDisabled();
		await fireEvent.input(document.getElementById('isds-username-test') as HTMLInputElement, {
			target: { value: 'u' }
		});
		expect(save).toBeDisabled();
		await fireEvent.input(document.getElementById('isds-password-test') as HTMLInputElement, {
			target: { value: 'p' }
		});
		expect(save).toBeEnabled();
	});

	it('shows a save error in an alert and keeps the form', async () => {
		mockFetch
			.mockResolvedValueOnce(jsonResponse(emptyOverview))
			.mockResolvedValueOnce(
				jsonResponse({ error: 'Systémová klíčenka není dostupná, proto se nic neuložilo.' }, 503)
			);
		render(ISDSCredentialsSection);
		await waitFor(() => expect(screen.getByTestId('isds-status-test')).toBeInTheDocument());

		await fireEvent.input(document.getElementById('isds-username-test') as HTMLInputElement, {
			target: { value: 'u' }
		});
		await fireEvent.input(document.getElementById('isds-password-test') as HTMLInputElement, {
			target: { value: 'p' }
		});
		await fireEvent.click(screen.getAllByRole('button', { name: 'Uložit přihlášení' })[0]);

		await waitFor(() => {
			expect(screen.getByRole('alert')).toHaveTextContent('Systémová klíčenka není dostupná');
		});
		expect(screen.getByTestId('isds-status-test')).toHaveTextContent('Přihlášení není uloženo');
	});

	it('asks for confirmation before deleting and sends the confirm header', async () => {
		mockFetch
			.mockResolvedValueOnce(jsonResponse(savedTestOverview))
			.mockResolvedValueOnce(new Response(null, { status: 204 }));
		render(ISDSCredentialsSection);
		await waitFor(() => {
			expect(screen.getByTestId('isds-status-test')).toHaveTextContent('Přihlášení uloženo');
		});

		await fireEvent.click(screen.getByRole('button', { name: 'Smazat přihlášení' }));
		expect(callsWith('DELETE')).toHaveLength(0);

		const dialog = await screen.findByRole('alertdialog');
		await fireEvent.click(within(dialog).getByRole('button', { name: 'Smazat' }));

		await waitFor(() => {
			expect(screen.getByTestId('isds-status-test')).toHaveTextContent('Přihlášení není uloženo');
		});
		const [url, init] = callsWith('DELETE')[0] as [string, RequestInit];
		expect(url).toContain('/api/v1/settings/isds-credentials?environment=test');
		expect((init.headers as Record<string, string>)['X-ZFaktury-Confirm']).toBe('1');
	});

	it('does not delete when the confirmation is cancelled', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(savedTestOverview));
		render(ISDSCredentialsSection);
		await waitFor(() => {
			expect(screen.getByTestId('isds-status-test')).toHaveTextContent('Přihlášení uloženo');
		});

		await fireEvent.click(screen.getByRole('button', { name: 'Smazat přihlášení' }));
		const dialog = await screen.findByRole('alertdialog');
		await fireEvent.click(within(dialog).getByRole('button', { name: 'Zrušit' }));

		expect(callsWith('DELETE')).toHaveLength(0);
		expect(screen.getByTestId('isds-status-test')).toHaveTextContent('Přihlášení uloženo');
	});

	describe('login test', () => {
		const okResult = {
			boxId: 'abc1234',
			boxType: 'FO',
			ownerName: 'Jan Testovaci',
			canCreateAndSend: true,
			canReadMessages: false
		};

		it('offers the test only for a configured environment and never runs it on load', async () => {
			mockFetch.mockResolvedValue(jsonResponse(savedTestOverview));
			render(ISDSCredentialsSection);
			await waitFor(() => {
				expect(screen.getByTestId('isds-status-test')).toHaveTextContent('Přihlášení uloženo');
			});

			expect(screen.getAllByRole('button', { name: 'Vyzkoušet přihlášení' })).toHaveLength(1);
			expect(callsWith('POST')).toHaveLength(0);
		});

		it('shows the box, owner and rights without a warning', async () => {
			mockFetch.mockResolvedValueOnce(jsonResponse(savedTestOverview));
			render(ISDSCredentialsSection);
			await waitFor(() => {
				expect(screen.getByTestId('isds-status-test')).toHaveTextContent('Přihlášení uloženo');
			});

			mockFetch.mockResolvedValueOnce(jsonResponse(okResult));
			await fireEvent.click(screen.getByRole('button', { name: 'Vyzkoušet přihlášení' }));

			const result = await screen.findByTestId('isds-test-result-test');
			expect(result).toHaveTextContent('abc1234');
			expect(result).toHaveTextContent('FO');
			expect(result).toHaveTextContent('Jan Testovaci');
			expect(result).toHaveTextContent('Smí odesílat');
			expect(within(result).queryByRole('alert')).not.toBeInTheDocument();

			const [url, init] = callsWith('POST')[0] as [string, RequestInit];
			expect(url).toContain('/api/v1/settings/isds-credentials/test');
			expect(JSON.parse(init.body as string)).toEqual({ environment: 'test' });
			expect((init.headers as Record<string, string>)['X-ZFaktury-Confirm']).toBe('1');
		});

		it('shows the warning prominently when the login can read delivered messages', async () => {
			mockFetch.mockResolvedValueOnce(jsonResponse(savedTestOverview));
			render(ISDSCredentialsSection);
			await waitFor(() => {
				expect(screen.getByTestId('isds-status-test')).toHaveTextContent('Přihlášení uloženo');
			});

			mockFetch.mockResolvedValueOnce(
				jsonResponse({
					...okResult,
					canReadMessages: true,
					warning: 'Tento účet smí číst doručené zprávy, přihlášení spustí doručení přihlášením.'
				})
			);
			await fireEvent.click(screen.getByRole('button', { name: 'Vyzkoušet přihlášení' }));

			const result = await screen.findByTestId('isds-test-result-test');
			expect(within(result).getByRole('alert')).toHaveTextContent(
				'přihlášení spustí doručení přihlášením'
			);
		});

		it('shows the server message when the test fails', async () => {
			mockFetch.mockResolvedValueOnce(jsonResponse(savedTestOverview));
			render(ISDSCredentialsSection);
			await waitFor(() => {
				expect(screen.getByTestId('isds-status-test')).toHaveTextContent('Přihlášení uloženo');
			});

			mockFetch.mockResolvedValueOnce(
				jsonResponse({ error: 'Datová schránka odmítla přihlašovací údaje.' }, 422)
			);
			await fireEvent.click(screen.getByRole('button', { name: 'Vyzkoušet přihlášení' }));

			await waitFor(() => {
				expect(screen.getByRole('alert')).toHaveTextContent(
					'Datová schránka odmítla přihlašovací údaje.'
				);
			});
			expect(screen.queryByTestId('isds-test-result-test')).not.toBeInTheDocument();
		});
	});

	it('shows a load error in an alert', async () => {
		mockFetch.mockResolvedValue(
			jsonResponse({ error: 'Systémová klíčenka není dostupná, proto nic nejde načíst.' }, 503)
		);
		render(ISDSCredentialsSection);

		await waitFor(() => {
			expect(screen.getByRole('alert')).toHaveTextContent('Systémová klíčenka není dostupná');
		});
	});
});
