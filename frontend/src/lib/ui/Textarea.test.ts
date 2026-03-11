import { describe, it, expect, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import Textarea from './Textarea.svelte';

afterEach(() => {
	cleanup();
});

describe('Textarea', () => {
	it('renders a textarea element', () => {
		render(Textarea);
		expect(document.querySelector('textarea')).toBeInTheDocument();
	});

	it('has consistent styling with Input component', () => {
		render(Textarea);
		const textarea = document.querySelector('textarea');
		expect(textarea?.className).toContain('rounded-lg');
		expect(textarea?.className).toContain('border-border');
		expect(textarea?.className).toContain('text-sm');
		expect(textarea?.className).toContain('focus:border-accent');
	});

	it('passes through HTML attributes', () => {
		render(Textarea, { props: { rows: 3, placeholder: 'Enter text...' } });
		const textarea = document.querySelector('textarea');
		expect(textarea?.getAttribute('rows')).toBe('3');
		expect(textarea?.getAttribute('placeholder')).toBe('Enter text...');
	});

	it('accepts custom class', () => {
		render(Textarea, { props: { class: 'custom-textarea' } });
		const textarea = document.querySelector('textarea');
		expect(textarea?.className).toContain('custom-textarea');
	});
});
