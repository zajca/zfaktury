import { describe, it, expect } from 'vitest';
import { ApiError, type FilingSubmissionStatus } from '$lib/api/client';
import {
	canRefresh,
	filingErrorMessage,
	findingIsWarning,
	findingLabel,
	findingPlace,
	formatFileSize,
	needsResolution,
	submissionStatusLabels,
	submissionStatusVariant
} from './filing';

const allStatuses: FilingSubmissionStatus[] = [
	'sending',
	'unknown',
	'sent',
	'delivered',
	'undeliverable',
	'failed'
];

describe('filing status module', () => {
	it('has a Czech label and a variant for every status', () => {
		for (const status of allStatuses) {
			expect(submissionStatusLabels[status]).toBeTruthy();
			expect(submissionStatusVariant[status]).toBeTruthy();
		}
	});

	it('asks for a resolution only for sending and unknown rows', () => {
		expect(allStatuses.filter(needsResolution)).toEqual(['sending', 'unknown']);
	});

	it('allows a refresh only for rows with a message id', () => {
		expect(allStatuses.filter(canRefresh)).toEqual(['sent', 'delivered', 'undeliverable']);
	});
});

describe('formatFileSize', () => {
	it('formats bytes, kilobytes and megabytes in Czech', () => {
		expect(formatFileSize(512)).toBe('512 B');
		expect(formatFileSize(1536)).toBe('1,5 kB');
		expect(formatFileSize(3 * 1024 * 1024)).toBe('3 MB');
	});
});

describe('filingErrorMessage', () => {
	it('returns the server message of an API error', () => {
		const err = new ApiError(409, 'Conflict', { error: 'Potvrzení vypršelo.' });
		expect(filingErrorMessage(err)).toBe('Potvrzení vypršelo.');
	});

	it('never exposes a raw status line', () => {
		const err = new ApiError(500, 'Internal Server Error');
		expect(filingErrorMessage(err)).toBe('Server vrátil neočekávanou odpověď.');
	});

	it('reports a transport failure in Czech', () => {
		expect(filingErrorMessage(new TypeError('Failed to fetch'))).toBe(
			'Nepodařilo se spojit se serverem.'
		);
	});
});

describe('finding helpers', () => {
	const finding = { type: 'k', code: '', field: 'c_ufo', section: 'VetaD', row: '3', text: 'x' };

	it('labels the finding type and falls back for an unknown one', () => {
		expect(findingLabel(finding)).toBe('Kritická chyba');
		expect(findingLabel({ ...finding, type: 'Z' })).toBe('Chyba');
	});

	it('tells warnings from errors', () => {
		expect(findingIsWarning(finding)).toBe(false);
		expect(findingIsWarning({ ...finding, type: 'P' })).toBe(true);
	});

	it('builds the place text', () => {
		expect(findingPlace(finding)).toBe('VetaD, c_ufo, řádek 3');
		expect(findingPlace({ ...finding, section: '', field: '', row: '' })).toBe('');
	});
});
