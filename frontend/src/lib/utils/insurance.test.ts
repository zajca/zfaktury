import { describe, it, expect } from 'vitest';
import { insuranceWarningText, formatMonthYear } from './insurance';

describe('insuranceWarningText', () => {
	it('names the payment year for the missing minimum', () => {
		expect(insuranceWarningText('advance_minimum_missing', 2026)).toBe(
			'Minimální záloha pro rok 2027 není v katalogu, ověřte ji u ČSSZ / pojišťovny'
		);
	});

	it('names the overview year for the missing filing method', () => {
		expect(insuranceWarningText('filing_method_missing', 2025)).toContain('za rok 2025');
	});

	it('explains an early filing', () => {
		expect(insuranceWarningText('advance_differs_from_xml', 2025)).toContain('podán dříve');
	});

	it('explains that health advances are not required', () => {
		const text = insuranceWarningText('health_advance_not_required', 2025);
		expect(text).toContain('v roce 2026 nejsou povinné');
		expect(text).toContain('za rok 2025');
	});

	it('returns an unknown token unchanged', () => {
		expect(insuranceWarningText('something_new', 2025)).toBe('something_new');
	});
});

describe('formatMonthYear', () => {
	it('formats month and year in Czech', () => {
		expect(formatMonthYear(6, 2026)).toBe('červen 2026');
		expect(formatMonthYear(12, 2026)).toBe('prosinec 2026');
	});
});
