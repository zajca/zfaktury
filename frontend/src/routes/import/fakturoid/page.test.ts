import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));
vi.mock('$app/environment', () => ({ browser: true }));

import { toasts, clearAllToasts } from '$lib/data/toast-state.svelte';
import Page from './+page.svelte';

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const importResult = {
	contacts_created: 3,
	contacts_skipped: 1,
	invoices_created: 5,
	invoices_skipped: 2,
	expenses_created: 4,
	expenses_skipped: 0,
	errors: []
};

beforeEach(() => {
	mockFetch.mockReset();
	clearAllToasts();
});

afterEach(() => {
	cleanup();
});

describe('Fakturoid import page', () => {
	it('shows the credentials form in idle state', () => {
		render(Page);

		expect(screen.getByLabelText('Slug účtu')).toBeInTheDocument();
		expect(screen.getByLabelText('Email')).toBeInTheDocument();
		expect(screen.getByLabelText('Client ID')).toBeInTheDocument();
		expect(screen.getByLabelText('Client Secret')).toBeInTheDocument();
		expect(screen.getByRole('button', { name: 'Importovat' })).toBeInTheDocument();
	});

	it('shows "Import dokončen" result after successful import', async () => {
		render(Page);

		// Fill in credentials
		await fireEvent.input(screen.getByLabelText('Slug účtu'), { target: { value: 'test-slug' } });
		await fireEvent.input(screen.getByLabelText('Email'), { target: { value: 'test@test.cz' } });
		await fireEvent.input(screen.getByLabelText('Client ID'), { target: { value: 'test-client-id' } });
		await fireEvent.input(screen.getByLabelText('Client Secret'), { target: { value: 'test-client-secret' } });

		// Remove required attrs to bypass HTML5 validation in jsdom
		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		// Mock import response
		mockFetch.mockResolvedValueOnce(jsonResponse(importResult));

		await fireEvent.click(screen.getByRole('button', { name: 'Importovat' }));

		await waitFor(() => {
			expect(screen.getByText('Import dokončen')).toBeInTheDocument();
		});

		// Check result stats
		expect(screen.getByText('Kontakty')).toBeInTheDocument();
		expect(screen.getByText('Faktury')).toBeInTheDocument();
		expect(screen.getByText('Náklady')).toBeInTheDocument();

		// Verify import API was called with correct endpoint and credentials
		expect(mockFetch).toHaveBeenCalledWith(
			'/api/v1/import/fakturoid/import',
			expect.objectContaining({ method: 'POST' })
		);
	});

	it('shows error when import fails', async () => {
		render(Page);

		await fireEvent.input(screen.getByLabelText('Slug účtu'), { target: { value: 'test-slug' } });
		await fireEvent.input(screen.getByLabelText('Email'), { target: { value: 'test@test.cz' } });
		await fireEvent.input(screen.getByLabelText('Client ID'), { target: { value: 'test-client-id' } });
		await fireEvent.input(screen.getByLabelText('Client Secret'), { target: { value: 'test-client-secret' } });

		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'Invalid credentials' }, 401));

		await fireEvent.click(screen.getByRole('button', { name: 'Importovat' }));

		await waitFor(() => {
			expect(toasts.some((t) => t.type === 'error')).toBe(true);
		});

		// Should return to idle state with form still visible
		expect(screen.getByLabelText('Slug účtu')).toBeInTheDocument();
	});

	it('shows import errors in result view', async () => {
		render(Page);

		await fireEvent.input(screen.getByLabelText('Slug účtu'), { target: { value: 'test-slug' } });
		await fireEvent.input(screen.getByLabelText('Email'), { target: { value: 'test@test.cz' } });
		await fireEvent.input(screen.getByLabelText('Client ID'), { target: { value: 'test-client-id' } });
		await fireEvent.input(screen.getByLabelText('Client Secret'), { target: { value: 'test-client-secret' } });

		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		const resultWithErrors = {
			...importResult,
			errors: ['contact 1: duplicate ICO', 'invoice 5: customer not resolved']
		};
		mockFetch.mockResolvedValueOnce(jsonResponse(resultWithErrors));

		await fireEvent.click(screen.getByRole('button', { name: 'Importovat' }));

		await waitFor(() => {
			expect(screen.getByText('Import dokončen')).toBeInTheDocument();
		});

		expect(screen.getByText('Chyby (2):')).toBeInTheDocument();
		expect(screen.getByText('contact 1: duplicate ICO')).toBeInTheDocument();
		expect(screen.getByText('invoice 5: customer not resolved')).toBeInTheDocument();
	});

	it('resets form when clicking "Nový import" after done', async () => {
		render(Page);

		await fireEvent.input(screen.getByLabelText('Slug účtu'), { target: { value: 'test-slug' } });
		await fireEvent.input(screen.getByLabelText('Email'), { target: { value: 'test@test.cz' } });
		await fireEvent.input(screen.getByLabelText('Client ID'), { target: { value: 'test-client-id' } });
		await fireEvent.input(screen.getByLabelText('Client Secret'), { target: { value: 'test-client-secret' } });

		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		mockFetch.mockResolvedValueOnce(jsonResponse(importResult));

		await fireEvent.click(screen.getByRole('button', { name: 'Importovat' }));

		await waitFor(() => {
			expect(screen.getByText('Import dokončen')).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByText('Nový import'));

		// Should show the form again with empty fields
		expect(screen.getByLabelText('Slug účtu')).toBeInTheDocument();
		expect((screen.getByLabelText('Slug účtu') as HTMLInputElement).value).toBe('');
	});

	it('disables form fields during import', async () => {
		render(Page);

		await fireEvent.input(screen.getByLabelText('Slug účtu'), { target: { value: 'test-slug' } });
		await fireEvent.input(screen.getByLabelText('Email'), { target: { value: 'test@test.cz' } });
		await fireEvent.input(screen.getByLabelText('Client ID'), { target: { value: 'test-client-id' } });
		await fireEvent.input(screen.getByLabelText('Client Secret'), { target: { value: 'test-client-secret' } });

		document.querySelectorAll('[required]').forEach((el) => el.removeAttribute('required'));

		// Use a promise that we control to keep the import "in progress"
		let resolveImport: (value: Response) => void;
		const importPromise = new Promise<Response>((resolve) => {
			resolveImport = resolve;
		});
		mockFetch.mockReturnValueOnce(importPromise);

		await fireEvent.click(screen.getByRole('button', { name: 'Importovat' }));

		await waitFor(() => {
			expect(screen.getByLabelText('Slug účtu')).toBeDisabled();
			expect(screen.getByLabelText('Email')).toBeDisabled();
			expect(screen.getByLabelText('Client ID')).toBeDisabled();
			expect(screen.getByLabelText('Client Secret')).toBeDisabled();
		});

		// Resolve the import
		resolveImport!(jsonResponse(importResult));

		await waitFor(() => {
			expect(screen.getByText('Import dokončen')).toBeInTheDocument();
		});
	});
});
