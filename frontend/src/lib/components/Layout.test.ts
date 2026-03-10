import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/svelte';
import LayoutTestWrapper from './LayoutTestWrapper.svelte';

vi.mock('$app/state', () => ({
	page: { params: {}, url: { pathname: '/invoices', searchParams: new URLSearchParams() } }
}));

afterEach(() => {
	cleanup();
});

describe('Layout', () => {
	it('renders navigation links', () => {
		render(LayoutTestWrapper);

		expect(screen.getByText('Faktury')).toBeInTheDocument();
		expect(screen.getByText('Kontakty')).toBeInTheDocument();
		expect(screen.getByText('Naklady')).toBeInTheDocument();
		expect(screen.getByText('Nastaveni')).toBeInTheDocument();
		expect(screen.getByText('Dashboard')).toBeInTheDocument();
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
		const aside = document.querySelector('aside');
		expect(aside?.className).toContain('-translate-x-full');
	});

	it('sidebar opens on toggle click', async () => {
		render(LayoutTestWrapper);
		const toggleBtn = screen.getByLabelText('Toggle menu');
		await fireEvent.click(toggleBtn);

		const aside = document.querySelector('aside');
		expect(aside?.className).toContain('translate-x-0');
	});

	it('sidebar closes on second toggle click', async () => {
		render(LayoutTestWrapper);
		const toggleBtn = screen.getByLabelText('Toggle menu');
		await fireEvent.click(toggleBtn);
		await fireEvent.click(toggleBtn);

		const aside = document.querySelector('aside');
		expect(aside?.className).toContain('-translate-x-full');
	});

	it('highlights active navigation item for /invoices', () => {
		render(LayoutTestWrapper);
		const invoicesLink = screen.getByText('Faktury').closest('a');
		expect(invoicesLink?.className).toContain('bg-blue-50');
		expect(invoicesLink?.className).toContain('text-blue-700');
	});

	it('does not highlight non-active navigation items', () => {
		render(LayoutTestWrapper);
		const contactsLink = screen.getByText('Kontakty').closest('a');
		expect(contactsLink?.className).not.toContain('bg-blue-50');
	});

	it('renders version info in footer', () => {
		render(LayoutTestWrapper);
		expect(screen.getByText('ZFaktury v0.1.0')).toBeInTheDocument();
	});

	it('navigation links have correct hrefs', () => {
		render(LayoutTestWrapper);

		const dashboardLink = screen.getByText('Dashboard').closest('a');
		expect(dashboardLink?.getAttribute('href')).toBe('/');

		const invoicesLink = screen.getByText('Faktury').closest('a');
		expect(invoicesLink?.getAttribute('href')).toBe('/invoices');

		const expensesLink = screen.getByText('Naklady').closest('a');
		expect(expensesLink?.getAttribute('href')).toBe('/expenses');

		const contactsLink = screen.getByText('Kontakty').closest('a');
		expect(contactsLink?.getAttribute('href')).toBe('/contacts');

		const settingsLink = screen.getByText('Nastaveni').closest('a');
		expect(settingsLink?.getAttribute('href')).toBe('/settings');
	});
});
