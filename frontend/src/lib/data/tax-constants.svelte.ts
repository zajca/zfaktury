import { taxConstantsApi, type TaxConstants } from '$lib/api/client';

const cache = new Map<number, TaxConstants>();

let current = $state<TaxConstants | null>(null);

export function getTaxConstants(): TaxConstants | null {
	return current;
}

export async function loadTaxConstants(year: number): Promise<TaxConstants | null> {
	const cached = cache.get(year);
	if (cached) {
		current = cached;
		return cached;
	}

	try {
		const tc = await taxConstantsApi.getByYear(year);
		cache.set(year, tc);
		current = tc;
		return tc;
	} catch {
		current = null;
		return null;
	}
}
