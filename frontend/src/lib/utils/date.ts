/**
 * Date formatting helpers for Czech locale.
 */

const czechDateFormatter = new Intl.DateTimeFormat('cs-CZ', {
	day: 'numeric',
	month: 'numeric',
	year: 'numeric'
});

const czechDateTimeFormatter = new Intl.DateTimeFormat('cs-CZ', {
	day: 'numeric',
	month: 'numeric',
	year: 'numeric',
	hour: '2-digit',
	minute: '2-digit'
});

const czechMonthYearFormatter = new Intl.DateTimeFormat('cs-CZ', {
	month: 'long',
	year: 'numeric'
});

/**
 * Check if a date value is valid and non-empty.
 */
function isValidDate(date: string | Date | null | undefined): date is string | Date {
	if (date == null) return false;
	if (typeof date === 'string' && date.trim() === '') return false;
	const d = typeof date === 'string' ? new Date(date) : date;
	return !isNaN(d.getTime());
}

/**
 * Format a date string or Date as Czech date: "1. 3. 2026"
 * Returns "-" for null, undefined, empty, or invalid dates.
 */
export function formatDate(date: string | Date | null | undefined): string {
	if (!isValidDate(date)) return '-';
	const d = typeof date === 'string' ? new Date(date) : date;
	return czechDateFormatter.format(d);
}

/**
 * Format a date string or Date as Czech datetime: "1. 3. 2026 14:30"
 * Returns "-" for null, undefined, empty, or invalid dates.
 */
export function formatDateTime(date: string | Date | null | undefined): string {
	if (!isValidDate(date)) return '-';
	const d = typeof date === 'string' ? new Date(date) : date;
	return czechDateTimeFormatter.format(d);
}

/**
 * Format as month and year: "brezen 2026"
 */
export function formatMonthYear(date: string | Date): string {
	const d = typeof date === 'string' ? new Date(date) : date;
	return czechMonthYearFormatter.format(d);
}

/**
 * Format a date as ISO date string (YYYY-MM-DD) suitable for input[type=date].
 */
export function toISODate(date: string | Date): string {
	const d = typeof date === 'string' ? new Date(date) : date;
	return d.toISOString().split('T')[0];
}

/**
 * Returns a human-readable relative time description in Czech.
 */
export function relativeDate(date: string | Date): string {
	const d = typeof date === 'string' ? new Date(date) : date;
	const now = new Date();
	const diffMs = d.getTime() - now.getTime();
	const diffDays = Math.round(diffMs / (1000 * 60 * 60 * 24));

	if (diffDays === 0) return 'dnes';
	if (diffDays === 1) return 'zitra';
	if (diffDays === -1) return 'vcera';
	if (diffDays > 1 && diffDays <= 7) return `za ${diffDays} dni`;
	if (diffDays < -1 && diffDays >= -7) return `pred ${Math.abs(diffDays)} dny`;

	return formatDate(d);
}
