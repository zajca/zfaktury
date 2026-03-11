import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import HelpTip from './HelpTip.svelte';

vi.mock('$app/environment', () => ({ browser: true }));

beforeEach(() => {
	vi.resetModules();
});

afterEach(() => {
	cleanup();
});

describe('HelpTip', () => {
	it('renders a button with question mark icon', () => {
		render(HelpTip, { props: { topic: 'duzp' } });
		const button = screen.getByRole('button');
		expect(button).toBeInTheDocument();
		expect(button.querySelector('svg')).toBeInTheDocument();
	});

	it('has correct aria-label with topic title', () => {
		render(HelpTip, { props: { topic: 'variabilni-symbol' } });
		const button = screen.getByRole('button');
		expect(button.getAttribute('aria-label')).toContain('Napoveda:');
		expect(button.getAttribute('aria-label')).toContain('Variabilni symbol');
	});

	it('has aria-haspopup="dialog"', () => {
		render(HelpTip, { props: { topic: 'ico' } });
		const button = screen.getByRole('button');
		expect(button.getAttribute('aria-haspopup')).toBe('dialog');
	});

	it('has cursor-help class', () => {
		render(HelpTip, { props: { topic: 'dic' } });
		const button = screen.getByRole('button');
		expect(button.className).toContain('cursor-help');
	});

	it('accepts custom class', () => {
		render(HelpTip, { props: { topic: 'iban', class: 'custom-class' } });
		const button = screen.getByRole('button');
		expect(button.className).toContain('custom-class');
	});

	it('has type="button" to prevent form submission', () => {
		render(HelpTip, { props: { topic: 'duzp' } });
		const button = screen.getByRole('button');
		expect(button.getAttribute('type')).toBe('button');
	});
});
