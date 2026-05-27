/**
 * Render and validate invoice number templates such as
 * "{prefix}-{yy}-{number:03d}". This is a TypeScript port of
 * internal/format/sequence.go and is exercised against the same fixture
 * (internal/format/testdata/render_cases.json) so the two implementations
 * cannot drift.
 */

const MIN_NUMBER_WIDTH = 1;
const MAX_NUMBER_WIDTH = 6;

/**
 * Render pattern against (prefix, year, number). Returns the formatted
 * invoice number. If the template is invalid, returns the raw pattern
 * unchanged so a programming error stays visible rather than silently
 * producing a half-rendered string. Mirrors the Go Render behaviour.
 */
export function renderSequence(pattern: string, prefix: string, year: number, number: number): string {
	if (validateSequencePattern(pattern) !== null) {
		return pattern;
	}
	let out = '';
	let i = 0;
	while (i < pattern.length) {
		if (pattern[i] !== '{') {
			out += pattern[i];
			i++;
			continue;
		}
		const end = pattern.indexOf('}', i);
		// validateSequencePattern already rejected unterminated braces.
		const token = pattern.substring(i + 1, end);
		out += renderToken(token, prefix, year, number);
		i = end + 1;
	}
	return out;
}

/**
 * Validate pattern. Returns null when valid, or a human-readable error
 * string when not. Callers should display a translated message; the Svelte
 * page maps "Neplatná šablona" + the raw string for now.
 */
export function validateSequencePattern(pattern: string): string | null {
	if (pattern.trim() === '') {
		return 'format pattern is empty';
	}
	let numberTokens = 0;
	let i = 0;
	while (i < pattern.length) {
		if (pattern[i] !== '{') {
			i++;
			continue;
		}
		const end = pattern.indexOf('}', i);
		if (end < 0) {
			return 'format pattern has unterminated "{"';
		}
		const token = pattern.substring(i + 1, end);
		const classification = classifyToken(token);
		if (classification.error !== null) {
			return classification.error;
		}
		if (classification.isNumber) {
			numberTokens++;
		}
		i = end + 1;
	}
	if (numberTokens !== 1) {
		return `format pattern must contain exactly one {number...} token, found ${numberTokens}`;
	}
	return null;
}

type Classification = { isNumber: boolean; error: string | null };

function classifyToken(token: string): Classification {
	switch (token) {
		case 'prefix':
			return { isNumber: false, error: null };
		case 'yyyy':
		case 'year':
			return { isNumber: false, error: null };
		case 'yy':
			return { isNumber: false, error: null };
		case 'number':
			return { isNumber: true, error: null };
	}
	if (token.startsWith('number:')) {
		const widthErr = parseNumberWidth(token.substring('number:'.length));
		if (widthErr.error !== null) {
			return { isNumber: false, error: widthErr.error };
		}
		return { isNumber: true, error: null };
	}
	return { isNumber: false, error: `unknown format token "{${token}}"` };
}

function parseNumberWidth(spec: string): { width: number; error: string | null } {
	if (!spec.endsWith('d')) {
		return { width: 0, error: `number width spec "${spec}" must end in 'd'` };
	}
	const digits = spec.substring(0, spec.length - 1);
	if (digits === '') {
		return { width: 0, error: 'number width spec is empty' };
	}
	if (!/^\d+$/.test(digits)) {
		return { width: 0, error: `number width "${digits}" is not numeric` };
	}
	const width = parseInt(digits, 10);
	if (width < MIN_NUMBER_WIDTH || width > MAX_NUMBER_WIDTH) {
		return { width: 0, error: `number width ${width} must be in ${MIN_NUMBER_WIDTH}..${MAX_NUMBER_WIDTH}` };
	}
	return { width, error: null };
}

function renderToken(token: string, prefix: string, year: number, number: number): string {
	switch (token) {
		case 'prefix':
			return prefix;
		case 'yyyy':
		case 'year':
			return String(year).padStart(4, '0');
		case 'yy':
			return String(year % 100).padStart(2, '0');
		case 'number':
			return String(number);
	}
	// Must be "number:Nd" -- validateSequencePattern already ensured it.
	const { width } = parseNumberWidth(token.substring('number:'.length));
	return String(number).padStart(width, '0');
}
