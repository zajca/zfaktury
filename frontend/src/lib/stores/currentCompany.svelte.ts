import { browser } from '$app/environment';
import type { Company } from '$lib/api/client';

const STORAGE_KEY = 'zfaktury.company';

let current = $state<Company | null>(null);
let companies = $state<Company[]>([]);

export const currentCompany = {
	get current() {
		return current;
	},
	get companies() {
		return companies;
	},

	setCompanies(list: Company[]) {
		companies = list;
		// If current is no longer in the list (e.g. soft-deleted), clear it.
		if (current && !list.find((c) => c.id === current!.id)) {
			current = null;
			if (browser) localStorage.removeItem(STORAGE_KEY);
		}
	},

	select(id: number) {
		const found = companies.find((c) => c.id === id);
		if (!found) return;
		current = found;
		if (browser) localStorage.setItem(STORAGE_KEY, String(id));
	},

	restoreSelection(): number | null {
		if (!browser) return null;
		const raw = localStorage.getItem(STORAGE_KEY);
		if (!raw) return null;
		const id = Number(raw);
		return Number.isFinite(id) && id > 0 ? id : null;
	},

	reset() {
		current = null;
		companies = [];
		if (browser) localStorage.removeItem(STORAGE_KEY);
	}
};
