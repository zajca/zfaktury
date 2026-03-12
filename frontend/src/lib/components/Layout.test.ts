import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import LayoutTestWrapper from './LayoutTestWrapper.svelte';

vi.mock('$app/state', () => ({
	page: { params: {}, url: { pathname: '/invoices', searchParams: new URLSearchParams() } }
}));

vi.mock('$app/environment', () => ({
	browser: false
}));

afterEach(() => {
	cleanup();
});

describe('Layout', () => {
	it('renders navigation links', () => {
		render(LayoutTestWrapper);

		// Desktop + mobile sidebars both have nav links, so use getAllByText
		expect(screen.getAllByText('Faktury').length).toBeGreaterThanOrEqual(1);
		expect(screen.getAllByText('Kontakty').length).toBeGreaterThanOrEqual(1);
		expect(screen.getAllByText('Náklady').length).toBeGreaterThanOrEqual(1);
		expect(screen.getAllByText('DPH').length).toBeGreaterThanOrEqual(1);
		expect(screen.getAllByText('Zálohy').length).toBeGreaterThanOrEqual(1);
		expect(screen.getAllByText('Nastavení').length).toBeGreaterThanOrEqual(1);
		expect(screen.getAllByText('Dashboard').length).toBeGreaterThanOrEqual(1);
	});

	it('renders children content', () => {
		render(LayoutTestWrapper);
		expect(screen.getByTestId('child-content')).toBeInTheDocument();
		expect(screen.getByText('Test child content')).toBeInTheDocument();
	});

	it('renders ZFaktury logo', () => {
		render(LayoutTestWrapper);
		const logos = screen.getAllByText('ZFaktury');
		expect(logos.length).toBeGreaterThanOrEqual(1);
	});

	it('has sidebar toggle button for mobile', () => {
		render(LayoutTestWrapper);
		const toggleBtn = screen.getByLabelText('Toggle menu');
		expect(toggleBtn).toBeInTheDocument();
	});

	it('sidebar starts closed (translated off-screen)', () => {
		render(LayoutTestWrapper);
		// Mobile sidebar (the one with lg:hidden)
		const asides = document.querySelectorAll('aside');
		const mobileSidebar = Array.from(asides).find((a) => a.className.includes('lg:hidden'));
		expect(mobileSidebar?.className).toContain('-translate-x-full');
	});

	it('sidebar opens on toggle click', async () => {
		render(LayoutTestWrapper);
		const toggleBtn = screen.getByLabelText('Toggle menu');
		await fireEvent.click(toggleBtn);

		const asides = document.querySelectorAll('aside');
		const mobileSidebar = Array.from(asides).find((a) => a.className.includes('lg:hidden'));
		expect(mobileSidebar?.className).toContain('translate-x-0');
	});

	it('sidebar closes on second toggle click', async () => {
		render(LayoutTestWrapper);
		const toggleBtn = screen.getByLabelText('Toggle menu');
		await fireEvent.click(toggleBtn);
		await fireEvent.click(toggleBtn);

		const asides = document.querySelectorAll('aside');
		const mobileSidebar = Array.from(asides).find((a) => a.className.includes('lg:hidden'));
		expect(mobileSidebar?.className).toContain('-translate-x-full');
	});

	it('highlights active navigation item for /invoices', () => {
		render(LayoutTestWrapper);
		const invoicesLinks = screen.getAllByText('Faktury');
		const invoicesLink = invoicesLinks[0].closest('a');
		expect(invoicesLink?.className).toContain('bg-accent-muted');
		expect(invoicesLink?.className).toContain('text-accent-text');
	});

	it('does not highlight non-active navigation items', () => {
		render(LayoutTestWrapper);
		const contactsLinks = screen.getAllByText('Kontakty');
		const contactsLink = contactsLinks[0].closest('a');
		expect(contactsLink?.className).not.toContain('bg-accent-muted');
	});

	it('renders version info in footer', () => {
		render(LayoutTestWrapper);
		const versions = screen.getAllByText('ZFaktury v0.1.0');
		expect(versions.length).toBeGreaterThanOrEqual(1);
	});

	it('renders section header for grouped navigation', () => {
		render(LayoutTestWrapper);
		const sections = screen.getAllByText('Účetnictví');
		expect(sections.length).toBeGreaterThanOrEqual(1);
	});

	it('navigation links have correct hrefs', () => {
		render(LayoutTestWrapper);

		const dashboardLink = screen.getAllByText('Dashboard')[0].closest('a');
		expect(dashboardLink?.getAttribute('href')).toBe('/');

		const invoicesLink = screen.getAllByText('Faktury')[0].closest('a');
		expect(invoicesLink?.getAttribute('href')).toBe('/invoices');

		const expensesLink = screen.getAllByText('Náklady')[0].closest('a');
		expect(expensesLink?.getAttribute('href')).toBe('/expenses');

		const vatLink = screen.getAllByText('DPH')[0].closest('a');
		expect(vatLink?.getAttribute('href')).toBe('/vat');

		const prepaymentsLink = screen.getAllByText('Zálohy')[0].closest('a');
		expect(prepaymentsLink?.getAttribute('href')).toBe('/tax/prepayments');

		const contactsLink = screen.getAllByText('Kontakty')[0].closest('a');
		expect(contactsLink?.getAttribute('href')).toBe('/contacts');

		const firmaLink = screen.getAllByText('Firma')[0].closest('a');
		expect(firmaLink?.getAttribute('href')).toBe('/settings/firma');

		const emailLink = screen.getAllByText('Email')[0].closest('a');
		expect(emailLink?.getAttribute('href')).toBe('/settings/email');
	});
});
