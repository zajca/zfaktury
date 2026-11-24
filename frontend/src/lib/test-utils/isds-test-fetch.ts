// Test helpers for the data-box send UI: synthetic fixtures and a fetch wrapper
// that answers the credentials and submission-list calls, so page tests can
// keep queueing their own responses for the page's calls.

import type { FilingPreview, FilingSubmission, ISDSCredentialsOverview } from '$lib/api/client';

export function jsonResponse(data: unknown, status = 200): Response {
	return new Response(JSON.stringify(data), {
		status,
		statusText: status === 200 ? 'OK' : 'Error',
		headers: { 'Content-Type': 'application/json' }
	});
}

export const noCredentials: ISDSCredentialsOverview = {
	test: { configured: false, username: '' },
	production: { configured: false, username: '' }
};

export const productionCredentials: ISDSCredentialsOverview = {
	test: { configured: false, username: '' },
	production: { configured: true, username: 'sender-user' }
};

export const testCredentials: ISDSCredentialsOverview = {
	test: { configured: true, username: 'sender-user' },
	production: { configured: false, username: '' }
};

export const bothCredentials: ISDSCredentialsOverview = {
	test: { configured: true, username: 'sender-test' },
	production: { configured: true, username: 'sender-prod' }
};

export function makePreview(overrides: Partial<FilingPreview> = {}): FilingPreview {
	return {
		kind: 'vat-return',
		documentId: 1,
		environment: 'production',
		recipientBoxId: '7nyn2d9',
		recipientName: 'Finanční úřad pro hlavní město Prahu',
		fileName: 'dph-2026-03.xml',
		annotation: 'Daňové přiznání k DPH za 3/2026',
		sizeBytes: 1536,
		sha256: 'a'.repeat(64),
		findings: [],
		requiresAcceptWarnings: false,
		confirmToken: 'token-1',
		expiresAt: '2026-03-10T12:05:00Z',
		...overrides
	};
}

export function makeSubmission(overrides: Partial<FilingSubmission> = {}): FilingSubmission {
	return {
		id: 1,
		kind: 'vat-return',
		documentId: 1,
		channel: 'isds',
		environment: 'production',
		recipientBoxId: '7nyn2d9',
		recipientName: 'Finanční úřad pro hlavní město Prahu',
		dmId: '1234567890',
		annotation: 'Daňové přiznání k DPH za 3/2026',
		fileName: 'dph-2026-03.xml',
		xmlSha256: 'a'.repeat(64),
		status: 'sent',
		dmMessageStatus: 2,
		sentAt: '2026-03-10T10:00:00Z',
		deliveredAt: null,
		hasReceipt: false,
		createdAt: '2026-03-10T10:00:00Z',
		...overrides
	};
}

export interface IsdsFetchState {
	credentials: ISDSCredentialsOverview;
	submissions: FilingSubmission[];
	/** Answers any other data-box call; return undefined to fall through to the fallback. */
	respond?: (url: string, init?: RequestInit) => Response | Promise<Response> | undefined;
}

type FetchFn = (url: string, init?: RequestInit) => Response | Promise<Response> | undefined;

/**
 * Builds a fetch replacement: the credentials and submission-list calls are
 * answered from `state`, everything else goes to `fallback`.
 */
export function createIsdsFetch(fallback: FetchFn, state: IsdsFetchState): typeof fetch {
	return ((input: RequestInfo | URL, init?: RequestInit) => {
		const url = String(input);
		const method = init?.method ?? 'GET';
		if (method === 'GET' && url.endsWith('/settings/isds-credentials')) {
			return Promise.resolve(jsonResponse(state.credentials));
		}
		if (method === 'GET' && /\/filings\/[a-z-]+\/\d+\/isds\/submissions$/.test(url)) {
			return Promise.resolve(jsonResponse(state.submissions));
		}
		if (url.includes('/filings/') || url.includes('/settings/isds-credentials')) {
			const answer = state.respond?.(url, init);
			if (answer) return Promise.resolve(answer);
		}
		return Promise.resolve(fallback(url, init));
	}) as typeof fetch;
}
