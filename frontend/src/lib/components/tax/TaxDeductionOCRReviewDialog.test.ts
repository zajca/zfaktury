import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import TaxDeductionOCRReviewDialog from './TaxDeductionOCRReviewDialog.svelte';
import type { TaxExtractionResult } from '$lib/api/client';

interface ConfirmedValues {
	category: string;
	description: string;
	provider_name: string;
	provider_ico: string;
	contract_number: string;
	document_date: string;
	purpose: string;
	claimed_amount_czk: number;
}

afterEach(() => {
	cleanup();
});

function makeResult(overrides: Partial<TaxExtractionResult> = {}): TaxExtractionResult {
	return {
		category: 'mortgage',
		provider_name: 'Česká spořitelna',
		provider_ico: '45244782',
		contract_number: '12345/2025',
		document_date: '2026-01-15',
		period_year: 2025,
		amount_czk: 45000,
		amount_halere: 4500000,
		purpose: '',
		description_suggestion: 'Úroky z hypotéky 2025',
		confidence: 0.92,
		...overrides
	};
}

describe('TaxDeductionOCRReviewDialog', () => {
	it('renders extracted provider, contract number, date and amount', () => {
		render(TaxDeductionOCRReviewDialog, {
			props: {
				result: makeResult(),
				onConfirm: vi.fn(),
				onCancel: vi.fn()
			}
		});

		expect(screen.getByLabelText('Poskytovatel')).toHaveValue('Česká spořitelna');
		expect(screen.getByLabelText('IČO poskytovatele')).toHaveValue('45244782');
		expect(screen.getByLabelText('Číslo smlouvy')).toHaveValue('12345/2025');
		expect(screen.getByLabelText('Datum dokladu')).toHaveValue('2026-01-15');
		expect(screen.getByLabelText('Uplatňovaná částka (CZK)')).toHaveValue(45000);
	});

	it('shows the confidence badge as a percentage', () => {
		render(TaxDeductionOCRReviewDialog, {
			props: {
				result: makeResult({ confidence: 0.73 }),
				onConfirm: vi.fn(),
				onCancel: vi.fn()
			}
		});

		expect(screen.getByTestId('confidence').textContent).toContain('73');
	});

	it('hides the purpose field unless category is donation or union_dues', async () => {
		render(TaxDeductionOCRReviewDialog, {
			props: {
				result: makeResult({ category: 'mortgage' }),
				onConfirm: vi.fn(),
				onCancel: vi.fn()
			}
		});
		expect(screen.queryByLabelText('Účel')).toBeNull();

		const select = screen.getByLabelText('Kategorie') as HTMLSelectElement;
		await fireEvent.change(select, { target: { value: 'donation' } });
		expect(screen.getByLabelText('Účel')).toBeTruthy();
	});

	it('calls onConfirm with edited values on submit', async () => {
		const onConfirm = vi.fn<(v: ConfirmedValues) => void>();
		render(TaxDeductionOCRReviewDialog, {
			props: {
				result: makeResult(),
				onConfirm,
				onCancel: vi.fn()
			}
		});

		const descInput = screen.getByLabelText('Popis') as HTMLInputElement;
		await fireEvent.input(descInput, { target: { value: 'Moje hypotéka 2025' } });

		const amountInput = screen.getByLabelText('Uplatňovaná částka (CZK)') as HTMLInputElement;
		await fireEvent.input(amountInput, { target: { value: '50000' } });

		const submit = screen.getByRole('button', { name: 'Uložit' });
		await fireEvent.click(submit);

		expect(onConfirm).toHaveBeenCalledTimes(1);
		const values = onConfirm.mock.calls[0][0];
		expect(values.category).toBe('mortgage');
		expect(values.description).toBe('Moje hypotéka 2025');
		expect(values.claimed_amount_czk).toBe(50000);
		expect(values.purpose).toBe('');
	});

	it('passes the purpose only when category requires it', async () => {
		const onConfirm = vi.fn<(v: ConfirmedValues) => void>();
		render(TaxDeductionOCRReviewDialog, {
			props: {
				result: makeResult({
					category: 'donation',
					purpose: 'Podpora charity',
					description_suggestion: 'Dar nemocnici'
				}),
				onConfirm,
				onCancel: vi.fn()
			}
		});

		const submit = screen.getByRole('button', { name: 'Uložit' });
		await fireEvent.click(submit);

		expect(onConfirm).toHaveBeenCalledTimes(1);
		expect(onConfirm.mock.calls[0][0].purpose).toBe('Podpora charity');
	});

	it('calls onCancel when the user clicks Zrušit', async () => {
		const onCancel = vi.fn();
		render(TaxDeductionOCRReviewDialog, {
			props: {
				result: makeResult(),
				onConfirm: vi.fn(),
				onCancel
			}
		});

		await fireEvent.click(screen.getByRole('button', { name: 'Zrušit' }));
		expect(onCancel).toHaveBeenCalledTimes(1);
	});

	it('falls back to amount_czk when amount_halere is zero', () => {
		render(TaxDeductionOCRReviewDialog, {
			props: {
				result: makeResult({ amount_czk: 12000, amount_halere: 0 }),
				onConfirm: vi.fn(),
				onCancel: vi.fn()
			}
		});

		expect(screen.getByLabelText('Uplatňovaná částka (CZK)')).toHaveValue(12000);
	});
});
