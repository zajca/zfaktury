import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import HelpDrawer from './HelpDrawer.svelte';
import { openHelp, closeHelp } from '$lib/data/help-state.svelte';

vi.mock('$app/environment', () => ({ browser: true }));

beforeEach(() => {
	closeHelp();
});

afterEach(() => {
	cleanup();
	closeHelp();
});

describe('HelpDrawer', () => {
	it('does not render dialog when closed', () => {
		render(HelpDrawer);
		expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
	});

	it('renders dialog when state is open before mount', () => {
		openHelp('duzp');
		render(HelpDrawer);
		expect(screen.getByRole('dialog')).toBeInTheDocument();
	});

	it('displays topic title in header', () => {
		openHelp('variabilni-symbol');
		render(HelpDrawer);
		expect(screen.getByText('Variabilni symbol')).toBeInTheDocument();
	});

	it('displays simple explanation section header', () => {
		openHelp('ico');
		render(HelpDrawer);
		expect(screen.getByText('Jednoduse')).toBeInTheDocument();
	});

	it('displays legal section header', () => {
		openHelp('ico');
		render(HelpDrawer);
		expect(screen.getByText('Pravni ramec')).toBeInTheDocument();
	});

	it('has close button with aria-label', () => {
		openHelp('duzp');
		render(HelpDrawer);
		expect(screen.getByLabelText('Zavrit napovedu')).toBeInTheDocument();
	});

	it('has aria-modal="true"', () => {
		openHelp('duzp');
		render(HelpDrawer);
		const dialog = screen.getByRole('dialog');
		expect(dialog.getAttribute('aria-modal')).toBe('true');
	});

	it('renders multiple paragraphs from simple content', () => {
		openHelp('duzp');
		render(HelpDrawer);
		const dialog = screen.getByRole('dialog');
		const paragraphs = dialog.querySelectorAll('.bg-elevated p');
		expect(paragraphs.length).toBeGreaterThan(1);
	});

	it('renders legal paragraphs', () => {
		openHelp('duzp');
		render(HelpDrawer);
		const dialog = screen.getByRole('dialog');
		// Legal section has paragraphs outside .bg-elevated
		const allParagraphs = dialog.querySelectorAll('p');
		expect(allParagraphs.length).toBeGreaterThan(2);
	});

	it('has mobile backdrop with role="presentation"', () => {
		openHelp('duzp');
		render(HelpDrawer);
		const backdrop = document.querySelector('[role="presentation"]');
		expect(backdrop).toBeInTheDocument();
	});
});
