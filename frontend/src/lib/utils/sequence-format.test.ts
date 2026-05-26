import { describe, it, expect } from 'vitest';
import { renderSequence, validateSequencePattern } from './sequence-format';
// Shared Go/TS fixture lives outside the frontend project — Vite resolves the
// relative path at test time and TS handles the import via resolveJsonModule.
import fixture from '../../../../internal/format/testdata/render_cases.json';

type RenderCase = {
	name: string;
	pattern: string;
	prefix: string;
	year: number;
	number: number;
	want: string;
};

type ValidationCase = {
	name: string;
	pattern: string;
};

type Fixture = {
	render_cases: RenderCase[];
	validation_errors: ValidationCase[];
};

const typed = fixture as Fixture;

describe('renderSequence (parity with Go)', () => {
	for (const tc of typed.render_cases) {
		it(tc.name, () => {
			expect(renderSequence(tc.pattern, tc.prefix, tc.year, tc.number)).toBe(tc.want);
		});
	}
});

describe('validateSequencePattern (parity with Go)', () => {
	for (const tc of typed.render_cases) {
		it(`valid: ${tc.name}`, () => {
			expect(validateSequencePattern(tc.pattern)).toBeNull();
		});
	}
	for (const tc of typed.validation_errors) {
		it(`invalid: ${tc.name}`, () => {
			const err = validateSequencePattern(tc.pattern);
			expect(err).not.toBeNull();
			expect(typeof err).toBe('string');
		});
	}
});
