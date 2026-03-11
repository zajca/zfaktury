import { describe, it, expect } from 'vitest';
import {
	formatDate,
	formatDateTime,
	formatMonthYear,
	toISODate,
	relativeDate,
	formatDateLong,
	addDays
} from './date';

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

	it('returns "-" for null', () => {
		expect(formatDate(null)).toBe('-');
	});

	it('returns "-" for undefined', () => {
		expect(formatDate(undefined)).toBe('-');
	});

	it('returns "-" for empty string', () => {
		expect(formatDate('')).toBe('-');
	});

	it('returns "-" for invalid date string', () => {
		expect(formatDate('not-a-date')).toBe('-');
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

	it('returns "-" for null', () => {
		expect(formatDateTime(null)).toBe('-');
	});

	it('returns "-" for undefined', () => {
		expect(formatDateTime(undefined)).toBe('-');
	});

	it('returns "-" for empty string', () => {
		expect(formatDateTime('')).toBe('-');
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

describe('formatDateLong', () => {
	it('formats a Date as long Czech date', () => {
		const d = new Date(2026, 2, 10); // March 10, 2026
		const result = formatDateLong(d);
		expect(result).toContain('10');
		expect(result).toContain('2026');
		// Should contain full Czech month name (e.g. "brezen" or "března")
		expect(result.length).toBeGreaterThan(10);
	});

	it('formats an ISO string', () => {
		const result = formatDateLong('2026-01-15');
		expect(result).toContain('15');
		expect(result).toContain('2026');
	});

	it('returns empty string for null', () => {
		expect(formatDateLong(null)).toBe('');
	});

	it('returns empty string for undefined', () => {
		expect(formatDateLong(undefined)).toBe('');
	});

	it('returns empty string for empty string', () => {
		expect(formatDateLong('')).toBe('');
	});

	it('returns empty string for invalid date', () => {
		expect(formatDateLong('not-a-date')).toBe('');
	});
});

describe('addDays', () => {
	it('adds positive days', () => {
		expect(addDays('2026-03-10', 7)).toBe('2026-03-17');
	});

	it('adds days across month boundary', () => {
		expect(addDays('2026-03-28', 5)).toBe('2026-04-02');
	});

	it('adds days across year boundary', () => {
		expect(addDays('2025-12-30', 3)).toBe('2026-01-02');
	});

	it('adds zero days', () => {
		expect(addDays('2026-03-10', 0)).toBe('2026-03-10');
	});

	it('subtracts days with negative value', () => {
		expect(addDays('2026-03-10', -5)).toBe('2026-03-05');
	});

	it('handles leap year', () => {
		expect(addDays('2024-02-28', 1)).toBe('2024-02-29');
		expect(addDays('2024-02-28', 2)).toBe('2024-03-01');
	});

	it('handles non-leap year', () => {
		expect(addDays('2026-02-28', 1)).toBe('2026-03-01');
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
