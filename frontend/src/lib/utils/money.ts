/**
 * Currency formatting helpers.
 * Amounts from the API are stored in halere (smallest unit, 1 CZK = 100 haleru).
 */

/**
 * Convert halere to crowns (float).
 */
export function fromHalere(halere: number): number {
	return halere / 100;
}

/**
 * Convert crowns (float) to halere (integer).
 */
export function toHalere(crowns: number): number {
	return Math.round(crowns * 100);
}

/**
 * Format an amount in halere as Czech Koruna string: "1 234,56 Kc".
 */
export function formatCZK(amountInHalere: number): string {
	const crowns = fromHalere(amountInHalere);
	return formatAmount(crowns, 'CZK');
}

/**
 * Format a numeric amount with the given currency symbol.
 */
export function formatAmount(amount: number, currency: string): string {
	const currencySymbols: Record<string, string> = {
		CZK: 'Kc',
		EUR: 'EUR',
		USD: 'USD'
	};

	const symbol = currencySymbols[currency] ?? currency;

	// Format with Czech locale (space as thousands separator, comma as decimal)
	const parts = Math.abs(amount).toFixed(2).split('.');
	const wholePart = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, '\u00A0');
	const formatted = `${wholePart},${parts[1]}`;

	const sign = amount < 0 ? '-' : '';
	return `${sign}${formatted}\u00A0${symbol}`;
}
