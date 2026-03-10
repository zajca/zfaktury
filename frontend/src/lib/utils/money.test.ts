import { describe, it, expect } from 'vitest';
import { fromHalere, toHalere, formatCZK, formatAmount } from './money';

describe('fromHalere', () => {
	it('converts zero', () => {
		expect(fromHalere(0)).toBe(0);
	});

	it('converts positive halere to crowns', () => {
		expect(fromHalere(10000)).toBe(100);
		expect(fromHalere(10050)).toBe(100.5);
		expect(fromHalere(1)).toBe(0.01);
	});

	it('converts negative halere', () => {
		expect(fromHalere(-2550)).toBe(-25.5);
	});
});

describe('toHalere', () => {
	it('converts zero', () => {
		expect(toHalere(0)).toBe(0);
	});

	it('converts crowns to halere', () => {
		expect(toHalere(100)).toBe(10000);
		expect(toHalere(100.5)).toBe(10050);
		expect(toHalere(0.01)).toBe(1);
	});

	it('rounds to nearest haler', () => {
		expect(toHalere(10.005)).toBe(1001);
		expect(toHalere(10.004)).toBe(1000);
	});

	it('converts negative amounts', () => {
		expect(toHalere(-25.5)).toBe(-2550);
	});
});

describe('fromHalere and toHalere roundtrip', () => {
	it('roundtrips integer amounts', () => {
		expect(toHalere(fromHalere(10050))).toBe(10050);
		expect(toHalere(fromHalere(0))).toBe(0);
		expect(toHalere(fromHalere(1))).toBe(1);
	});
});

describe('formatCZK', () => {
	it('formats zero', () => {
		expect(formatCZK(0)).toBe('0,00\u00A0Kc');
	});

	it('formats small amounts', () => {
		expect(formatCZK(100)).toBe('1,00\u00A0Kc');
		expect(formatCZK(1)).toBe('0,01\u00A0Kc');
	});

	it('formats large amounts with thousands separator', () => {
		expect(formatCZK(123456789)).toBe('1\u00A0234\u00A0567,89\u00A0Kc');
	});

	it('formats negative amounts', () => {
		expect(formatCZK(-2550)).toBe('-25,50\u00A0Kc');
	});
});

describe('formatAmount', () => {
	it('formats with CZK symbol', () => {
		expect(formatAmount(100, 'CZK')).toBe('100,00\u00A0Kc');
	});

	it('formats with EUR symbol', () => {
		expect(formatAmount(100, 'EUR')).toBe('100,00\u00A0EUR');
	});

	it('formats with USD symbol', () => {
		expect(formatAmount(50.5, 'USD')).toBe('50,50\u00A0USD');
	});

	it('uses currency code for unknown currencies', () => {
		expect(formatAmount(100, 'GBP')).toBe('100,00\u00A0GBP');
	});

	it('formats thousands with non-breaking space', () => {
		expect(formatAmount(1234567.89, 'CZK')).toBe('1\u00A0234\u00A0567,89\u00A0Kc');
	});

	it('formats negative amounts with sign', () => {
		expect(formatAmount(-100, 'CZK')).toBe('-100,00\u00A0Kc');
	});
});
