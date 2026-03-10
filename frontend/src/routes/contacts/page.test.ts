import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, cleanup } from '@testing-library/svelte';
import Page from './+page.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const sampleContacts = {
	data: [
		{
			id: 1,
			name: 'Test Corp',
			ico: '12345678',
			dic: 'CZ12345678',
			city: 'Praha',
			email: 'test@test.cz',
			phone: '+420123456789',
			type: 'company'
		},
		{
			id: 2,
			name: 'Jan Novak',
			ico: '',
			dic: '',
			city: 'Brno',
			email: 'jan@novak.cz',
			phone: '',
			type: 'individual'
		}
	],
	total: 2,
	limit: 25,
	offset: 0
};

const emptyContacts = {
	data: [],
	total: 0,
	limit: 25,
	offset: 0
};

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('Contacts list page', () => {
	it('loads contacts on mount', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalled();
		});

		const url = mockFetch.mock.calls[0][0] as string;
		expect(url).toContain('/api/v1/contacts');
	});

	it('renders contact rows with name, ICO, and city', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleContacts));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Test Corp')).toBeInTheDocument();
		});

		expect(screen.getByText('Jan Novak')).toBeInTheDocument();
		expect(screen.getByText('12345678')).toBeInTheDocument();
		expect(screen.getByText('Praha')).toBeInTheDocument();
		expect(screen.getByText('Brno')).toBeInTheDocument();
	});

	it('shows empty state message when no contacts', async () => {
		mockFetch.mockResolvedValue(jsonResponse(emptyContacts));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Zatím žádné kontakty.')).toBeInTheDocument();
		});
	});

	it('shows error state on API failure', async () => {
		mockFetch.mockRejectedValue(new Error('Network error'));

		render(Page);

		await waitFor(() => {
			expect(screen.getByText('Network error')).toBeInTheDocument();
		});
	});

	it('has a search input', async () => {
		mockFetch.mockResolvedValue(jsonResponse(sampleContacts));

		render(Page);

		const searchInput = screen.getByPlaceholderText('Hledat podle názvu, IČO, emailu...');
		expect(searchInput).toBeInTheDocument();
	});
});
