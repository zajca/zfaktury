import { describe, it, expect } from 'vitest';
import { formatDate, formatDateTime, formatMonthYear, toISODate, relativeDate } from './date';

describe('formatDate', () => {
	it('formats a Date object', () => {
		const d = new Date(2026, 2, 10); // March 10, 2026
		const result = formatDate(d);
		// Czech locale: "10. 3. 2026"
		expect(result).toContain('10');
		expect(result).toContain('3');
		expect(result).toContain('2026');
	});

	it('formats an ISO date string', () => {
		const result = formatDate('2026-01-15');
		expect(result).toContain('15');
		expect(result).toContain('1');
		expect(result).toContain('2026');
	});
});

describe('formatDateTime', () => {
	it('includes time in output', () => {
		const d = new Date(2026, 2, 10, 14, 30);
		const result = formatDateTime(d);
		expect(result).toContain('10');
		expect(result).toContain('14');
		expect(result).toContain('30');
	});
});

describe('formatMonthYear', () => {
	it('formats month and year in Czech', () => {
		const d = new Date(2026, 2, 10); // March 2026
		const result = formatMonthYear(d);
		expect(result).toContain('2026');
		// Czech "brezen" or similar month name
		expect(result.length).toBeGreaterThan(4);
	});
});

describe('toISODate', () => {
	it('converts Date to ISO format', () => {
		// Use UTC noon to avoid timezone shifts
		const d = new Date('2026-03-10T12:00:00Z');
		expect(toISODate(d)).toBe('2026-03-10');
	});

	it('converts ISO string to ISO date (strips time)', () => {
		expect(toISODate('2026-03-10T14:30:00Z')).toBe('2026-03-10');
	});

	it('handles date string input', () => {
		expect(toISODate('2026-01-01')).toBe('2026-01-01');
	});
});

describe('relativeDate', () => {
	it('returns "dnes" for today', () => {
		const now = new Date();
		expect(relativeDate(now)).toBe('dnes');
	});

	it('returns "zitra" for tomorrow', () => {
		const tomorrow = new Date();
		tomorrow.setDate(tomorrow.getDate() + 1);
		expect(relativeDate(tomorrow)).toBe('zitra');
	});

	it('returns "vcera" for yesterday', () => {
		const yesterday = new Date();
		yesterday.setDate(yesterday.getDate() - 1);
		expect(relativeDate(yesterday)).toBe('vcera');
	});

	it('returns "za N dni" for near future', () => {
		const future = new Date();
		future.setDate(future.getDate() + 3);
		expect(relativeDate(future)).toBe('za 3 dni');
	});

	it('returns "pred N dny" for near past', () => {
		const past = new Date();
		past.setDate(past.getDate() - 5);
		expect(relativeDate(past)).toBe('pred 5 dny');
	});

	it('returns formatted date for distant dates', () => {
		const distant = new Date(2025, 0, 1);
		const result = relativeDate(distant);
		// Should fall back to formatDate
		expect(result).toContain('2025');
	});
});
