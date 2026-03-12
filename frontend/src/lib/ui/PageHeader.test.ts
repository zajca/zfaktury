import { describe, it, expect, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import PageHeader from './PageHeader.svelte';

afterEach(() => {
	cleanup();
});

describe('PageHeader', () => {
	it('renders title', () => {
		render(PageHeader, { props: { title: 'Faktury' } });
		expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Faktury');
	});

	it('renders description when provided', () => {
		render(PageHeader, { props: { title: 'Faktury', description: 'Přehled faktur' } });
		expect(screen.getByText('Přehled faktur')).toBeInTheDocument();
	});

	it('does not render description when not provided', () => {
		const { container } = render(PageHeader, { props: { title: 'Faktury' } });
		expect(container.querySelector('p')).not.toBeInTheDocument();
	});

	it('renders back link when backHref is provided', () => {
		render(PageHeader, {
			props: { title: 'Detail', backHref: '/invoices', backLabel: 'Zpět na faktury' }
		});
		const link = screen.getByText(/Zpět na faktury/);
		expect(link).toBeInTheDocument();
		expect(link.getAttribute('href')).toBe('/invoices');
	});

	it('uses default back label when backLabel is not provided', () => {
		render(PageHeader, { props: { title: 'Detail', backHref: '/invoices' } });
		expect(screen.getByText(/Zpět/)).toBeInTheDocument();
	});

	it('does not render back link when backHref is not provided', () => {
		const { container } = render(PageHeader, { props: { title: 'Faktury' } });
		const links = container.querySelectorAll('a');
		expect(links.length).toBe(0);
	});

	it('accepts custom class', () => {
		const { container } = render(PageHeader, { props: { title: 'Test', class: 'mb-6' } });
		expect(container.firstElementChild?.className).toContain('mb-6');
	});
});
