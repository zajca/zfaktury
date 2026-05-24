import { describe, it, expect, beforeEach } from 'vitest';
import { currentCompany } from './currentCompany.svelte';
import type { Company } from '$lib/api/client';

const A: Company = {
	id: 1,
	name: 'A',
	legal_name: 'A',
	ico: '1',
	vat_registered: false,
	created_at: '',
	updated_at: ''
};
const B: Company = {
	id: 2,
	name: 'B',
	legal_name: 'B',
	ico: '2',
	vat_registered: false,
	created_at: '',
	updated_at: ''
};

beforeEach(() => {
	localStorage.clear();
	currentCompany.reset();
});

describe('currentCompany store', () => {
	it('starts empty', () => {
		expect(currentCompany.current).toBeNull();
		expect(currentCompany.companies).toEqual([]);
	});

	it('setCompanies populates the list', () => {
		currentCompany.setCompanies([A, B]);
		expect(currentCompany.companies).toHaveLength(2);
	});

	it('select sets current and persists to localStorage', () => {
		currentCompany.setCompanies([A, B]);
		currentCompany.select(2);
		expect(currentCompany.current?.id).toBe(2);
		expect(localStorage.getItem('zfaktury.company')).toBe('2');
	});

	it('restoreSelection returns id from localStorage', () => {
		localStorage.setItem('zfaktury.company', '2');
		expect(currentCompany.restoreSelection()).toBe(2);
	});

	it('restoreSelection returns null when nothing stored', () => {
		expect(currentCompany.restoreSelection()).toBeNull();
	});

	it('select with an unknown id leaves current null', () => {
		currentCompany.setCompanies([A]);
		currentCompany.select(99);
		expect(currentCompany.current).toBeNull();
	});

	it('setCompanies clears current if it is no longer in the list', () => {
		currentCompany.setCompanies([A, B]);
		currentCompany.select(2);
		currentCompany.setCompanies([A]); // B disappears
		expect(currentCompany.current).toBeNull();
	});
});
