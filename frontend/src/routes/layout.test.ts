import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, waitFor, cleanup } from '@testing-library/svelte';
import { currentCompany } from '$lib/stores/currentCompany.svelte';
import type { Company } from '$lib/api/client';
import LayoutWrapper from './layout-test-wrapper.svelte';

// --- Mocks ---

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

vi.mock('$app/state', () => ({
	page: { url: { pathname: '/', searchParams: new URLSearchParams() }, params: {} }
}));

vi.mock('$lib/utils/download', () => ({
	isDesktopMode: vi.fn().mockResolvedValue(false),
	downloadFile: vi.fn()
}));

// Mock the companies endpoint at the API client level so we don't have to
// stub global fetch (the inner Layout component also imports the API client
// transitively).
vi.mock('$lib/api/client', async (original) => {
	const actual = (await original()) as Record<string, unknown>;
	return {
		...actual,
		companiesApi: {
			list: vi.fn(),
			getById: vi.fn(),
			create: vi.fn(),
			update: vi.fn(),
			delete: vi.fn()
		}
	};
});

// --- Fixtures ---

const A: Company = {
	id: 1,
	name: 'Firma A',
	legal_name: 'Firma A s.r.o.',
	ico: '11111111',
	vat_registered: false,
	created_at: '',
	updated_at: ''
};

const B: Company = {
	id: 2,
	name: 'Firma B',
	legal_name: 'Firma B s.r.o.',
	ico: '22222222',
	vat_registered: false,
	created_at: '',
	updated_at: ''
};

beforeEach(async () => {
	// Wipe the store, localStorage, and goto/api mocks so each bootstrap
	// scenario starts from a known empty state.
	currentCompany.reset();
	localStorage.clear();
	const { goto } = await import('$app/navigation');
	vi.mocked(goto).mockClear();
	const { companiesApi } = await import('$lib/api/client');
	vi.mocked(companiesApi.list).mockReset();
});

afterEach(() => {
	cleanup();
});

describe('+layout bootstrap', () => {
	it('shows the loading spinner before bootstrap resolves', async () => {
		const { companiesApi } = await import('$lib/api/client');
		// Never resolve — the spinner should stay visible.
		vi.mocked(companiesApi.list).mockReturnValue(new Promise(() => {}));

		const { container } = render(LayoutWrapper);

		const status = container.querySelector('[role="status"]');
		expect(status).toBeTruthy();
		expect(status?.textContent).toContain('Načítání');
	});

	it('redirects to /companies/new when the list is empty', async () => {
		const { companiesApi } = await import('$lib/api/client');
		vi.mocked(companiesApi.list).mockResolvedValue([]);
		const { goto } = await import('$app/navigation');

		render(LayoutWrapper);

		await waitFor(() => {
			expect(goto).toHaveBeenCalledWith('/companies/new');
		});
	});

	it('does NOT redirect when already on /companies/new with empty list', async () => {
		const { page } = await import('$app/state');
		(page as { url: { pathname: string } }).url.pathname = '/companies/new';

		const { companiesApi } = await import('$lib/api/client');
		vi.mocked(companiesApi.list).mockResolvedValue([]);
		const { goto } = await import('$app/navigation');

		render(LayoutWrapper);

		// Give the onMount microtask a chance to run.
		await new Promise((r) => setTimeout(r, 0));
		expect(goto).not.toHaveBeenCalled();

		// Reset pathname for other tests.
		(page as { url: { pathname: string } }).url.pathname = '/';
	});

	it('selects the first company when localStorage is empty', async () => {
		const { companiesApi } = await import('$lib/api/client');
		vi.mocked(companiesApi.list).mockResolvedValue([A, B]);

		render(LayoutWrapper);

		await waitFor(() => {
			expect(currentCompany.current?.id).toBe(1);
		});
	});

	it('restores the previously-active company from localStorage', async () => {
		// Pre-seed localStorage as if the user had selected company B last session.
		// We have to do this in a way that survives currentCompany.reset() above —
		// reset() clears localStorage, so we seed AFTER beforeEach by writing
		// before render.
		localStorage.setItem('zfaktury.company', '2');

		const { companiesApi } = await import('$lib/api/client');
		vi.mocked(companiesApi.list).mockResolvedValue([A, B]);

		render(LayoutWrapper);

		await waitFor(() => {
			expect(currentCompany.current?.id).toBe(2);
		});
	});

	it('falls back to first company when localStorage points to a missing id', async () => {
		localStorage.setItem('zfaktury.company', '999');

		const { companiesApi } = await import('$lib/api/client');
		vi.mocked(companiesApi.list).mockResolvedValue([A, B]);

		render(LayoutWrapper);

		await waitFor(() => {
			expect(currentCompany.current?.id).toBe(1);
		});
	});

	it('keeps the spinner gone (booted=true) even if the API call fails', async () => {
		const { companiesApi } = await import('$lib/api/client');
		vi.mocked(companiesApi.list).mockRejectedValue(new Error('boom'));

		// Silence the expected console.error so test output stays clean.
		const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

		const { container } = render(LayoutWrapper);

		await waitFor(() => {
			expect(container.querySelector('[role="status"]')).toBeNull();
		});

		errorSpy.mockRestore();
	});
});
