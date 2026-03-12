import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import CurrencySelector from './CurrencySelector.svelte';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

function jsonResponse(data: unknown, status = 200) {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

const mockRateResult = {
	currency_code: 'EUR',
	rate: 25.32,
	date: '2026-03-11'
};

beforeEach(() => {
	mockFetch.mockReset();
});

afterEach(() => {
	cleanup();
});

describe('CurrencySelector', () => {
	it('renders with CZK selected by default, no rate field visible', () => {
		render(CurrencySelector, { props: { currency: 'CZK', exchangeRate: 1 } });

		const select = screen.getByLabelText('Měna') as HTMLSelectElement;
		expect(select).toBeInTheDocument();
		expect(select.value).toBe('CZK');

		// Rate input should not be visible for CZK
		expect(screen.queryByLabelText('Směnný kurz')).not.toBeInTheDocument();
		expect(screen.queryByText('Načíst kurz')).not.toBeInTheDocument();
	});

	it('renders all currency options', () => {
		render(CurrencySelector, { props: { currency: 'CZK', exchangeRate: 1 } });

		const options = screen.getByLabelText('Měna').querySelectorAll('option');
		const values = Array.from(options).map((o) => o.value);
		expect(values).toEqual(['CZK', 'EUR', 'USD', 'GBP', 'PLN', 'CHF']);
	});

	it('shows rate field when EUR is selected', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(mockRateResult));

		render(CurrencySelector, { props: { currency: 'CZK', exchangeRate: 1 } });

		const select = screen.getByLabelText('Měna') as HTMLSelectElement;
		await fireEvent.change(select, { target: { value: 'EUR' } });

		await waitFor(() => {
			expect(screen.getByLabelText('Směnný kurz')).toBeInTheDocument();
			expect(screen.getByText('Načíst kurz')).toBeInTheDocument();
		});
	});

	it('fetches rate from API on currency change', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(mockRateResult));

		render(CurrencySelector, { props: { currency: 'CZK', exchangeRate: 1 } });

		const select = screen.getByLabelText('Měna') as HTMLSelectElement;
		await fireEvent.change(select, { target: { value: 'EUR' } });

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledWith(
				expect.stringContaining('/api/v1/exchange-rate?currency=EUR'),
				expect.any(Object)
			);
		});

		// Should display the rate source
		await waitFor(() => {
			expect(screen.getByText('CNB, 2026-03-11')).toBeInTheDocument();
		});
	});

	it('displays error on API failure', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse({ error: 'not found' }, 500));

		render(CurrencySelector, { props: { currency: 'CZK', exchangeRate: 1 } });

		const select = screen.getByLabelText('Měna') as HTMLSelectElement;
		await fireEvent.change(select, { target: { value: 'EUR' } });

		await waitFor(() => {
			const errorEl = screen.getByRole('alert');
			expect(errorEl).toBeInTheDocument();
			expect(errorEl.textContent).toContain('Nepodařilo se načíst kurz');
		});
	});

	it('hides rate field when switching back to CZK', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(mockRateResult));

		render(CurrencySelector, { props: { currency: 'EUR', exchangeRate: 25.32 } });

		// Wait for initial fetch
		await waitFor(() => {
			expect(screen.getByLabelText('Směnný kurz')).toBeInTheDocument();
		});

		const select = screen.getByLabelText('Měna') as HTMLSelectElement;
		await fireEvent.change(select, { target: { value: 'CZK' } });

		await waitFor(() => {
			expect(screen.queryByLabelText('Směnný kurz')).not.toBeInTheDocument();
		});
	});

	it('manual refetch button triggers API call', async () => {
		mockFetch
			.mockResolvedValueOnce(jsonResponse(mockRateResult))
			.mockResolvedValueOnce(jsonResponse({ ...mockRateResult, rate: 25.5 }));

		render(CurrencySelector, { props: { currency: 'CZK', exchangeRate: 1 } });

		const select = screen.getByLabelText('Měna') as HTMLSelectElement;
		await fireEvent.change(select, { target: { value: 'EUR' } });

		await waitFor(() => {
			expect(screen.getByText('Načíst kurz')).toBeInTheDocument();
		});

		const refetchBtn = screen.getByText('Načíst kurz');
		await fireEvent.click(refetchBtn);

		await waitFor(() => {
			expect(mockFetch).toHaveBeenCalledTimes(2);
		});
	});

	it('calls onchange callback when currency changes', async () => {
		const onchangeSpy = vi.fn();
		mockFetch.mockResolvedValueOnce(jsonResponse(mockRateResult));

		render(CurrencySelector, {
			props: { currency: 'CZK', exchangeRate: 1, onchange: onchangeSpy }
		});

		const select = screen.getByLabelText('Měna') as HTMLSelectElement;
		await fireEvent.change(select, { target: { value: 'EUR' } });

		await waitFor(() => {
			expect(onchangeSpy).toHaveBeenCalledWith('EUR', 25.32);
		});
	});

	it('allows manual rate editing', async () => {
		mockFetch.mockResolvedValueOnce(jsonResponse(mockRateResult));
		const onchangeSpy = vi.fn();

		render(CurrencySelector, {
			props: { currency: 'EUR', exchangeRate: 25.32, onchange: onchangeSpy }
		});

		await waitFor(() => {
			expect(screen.getByLabelText('Směnný kurz')).toBeInTheDocument();
		});

		const rateInput = screen.getByLabelText('Směnný kurz') as HTMLInputElement;
		// Simulate user typing a new rate
		Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.call(
			rateInput,
			'26.00'
		);
		await fireEvent.input(rateInput, { target: { value: '26.00' } });

		expect(onchangeSpy).toHaveBeenCalledWith('EUR', 26.0);
	});
});
