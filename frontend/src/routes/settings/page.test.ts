import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import Page from './+page.svelte';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

beforeEach(() => {
	vi.clearAllMocks();
});

afterEach(() => {
	cleanup();
});

describe('Settings Page (redirect)', () => {
	it('redirects to /settings/firma on mount', async () => {
		render(Page);
		const { goto } = await import('$app/navigation');
		expect(goto).toHaveBeenCalledWith('/settings/firma', { replaceState: true });
	});
});
