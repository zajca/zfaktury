// Shared labels, colors and helpers for sending filings through the data box.

import {
	ApiError,
	type EPOFinding,
	type FilingSubmissionStatus,
	type ISDSEnvironment
} from '$lib/api/client';

export type FilingVariant = 'default' | 'info' | 'success' | 'danger' | 'warning' | 'muted';

export const submissionStatusLabels: Record<FilingSubmissionStatus, string> = {
	sending: 'Odesílá se',
	unknown: 'Výsledek není znám',
	sent: 'Odesláno',
	delivered: 'Doručeno',
	undeliverable: 'Nedoručitelné',
	failed: 'Neodesláno'
};

export const submissionStatusVariant: Record<FilingSubmissionStatus, FilingVariant> = {
	sending: 'warning',
	unknown: 'warning',
	sent: 'info',
	delivered: 'success',
	undeliverable: 'danger',
	failed: 'danger'
};

export const environmentLabels: Record<ISDSEnvironment, string> = {
	production: 'Produkce',
	test: 'Testovací prostředí'
};

export const environmentVariant: Record<ISDSEnvironment, FilingVariant> = {
	production: 'danger',
	test: 'info'
};

/** Environments in the order they are offered to the user. */
export const environmentOrder: ISDSEnvironment[] = ['production', 'test'];

/** Statuses whose outcome the user has to settle by hand. */
export function needsResolution(status: FilingSubmissionStatus): boolean {
	return status === 'unknown' || status === 'sending';
}

/** Statuses for which ISDS can be asked for the delivery state. */
export function canRefresh(status: FilingSubmissionStatus): boolean {
	return status === 'sent' || status === 'delivered' || status === 'undeliverable';
}

const sizeFormatter = new Intl.NumberFormat('cs-CZ', { maximumFractionDigits: 1 });

/** Formats a byte count in Czech, e.g. "1,5 kB". */
export function formatFileSize(bytes: number): string {
	if (bytes < 1024) return `${sizeFormatter.format(bytes)} B`;
	if (bytes < 1024 * 1024) return `${sizeFormatter.format(bytes / 1024)} kB`;
	return `${sizeFormatter.format(bytes / (1024 * 1024))} MB`;
}

const MSG_NETWORK = 'Nepodařilo se spojit se serverem.';
const MSG_UNKNOWN = 'Server vrátil neočekávanou odpověď.';

/** Czech text of a failed API call: the server's message, never a raw status line. */
export function filingErrorMessage(e: unknown): string {
	if (e instanceof ApiError) {
		const body = e.body;
		if (body && typeof body === 'object' && 'error' in body) {
			const text = (body as { error: unknown }).error;
			if (typeof text === 'string' && text !== '') return text;
		}
		return MSG_UNKNOWN;
	}
	return MSG_NETWORK;
}

const findingLabels: Record<string, string> = {
	K: 'Kritická chyba',
	S: 'Chyba struktury',
	E: 'Chyba EPO',
	P: 'Varování',
	N: 'Upozornění',
	I: 'Informace'
};

export function findingLabel(finding: EPOFinding): string {
	return findingLabels[finding.type.trim().toUpperCase()] ?? 'Chyba';
}

export function findingIsWarning(finding: EPOFinding): boolean {
	return ['P', 'N', 'I'].includes(finding.type.trim().toUpperCase());
}

export function findingPlace(finding: EPOFinding): string {
	const parts = [finding.section, finding.field];
	if (finding.row) parts.push(`řádek ${finding.row}`);
	return parts.filter(Boolean).join(', ');
}
