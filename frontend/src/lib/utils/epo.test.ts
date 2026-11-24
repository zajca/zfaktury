import { describe, it, expect } from 'vitest';
import { isEPOFormUrl } from './epo';

describe('isEPOFormUrl', () => {
	it('accepts an https EPO form link', () => {
		expect(
			isEPOFormUrl('https://mojedane.gov.cz/dpr/adis/idpr_epo/epo2/form/form_uvod.faces?form_id=X')
		).toBe(true);
	});

	it.each([
		['javascript:alert(1)'],
		['data:text/html,<p>x</p>'],
		['http://mojedane.gov.cz/dpr/x'],
		['https://evil.example/dpr/x'],
		['https://mojedane.gov.cz.evil.example/dpr/x'],
		['https://mojedane.gov.cz:8443/dpr/x'],
		['https://user:pw@mojedane.gov.cz/dpr/x'],
		['https://mojedane.gov.cz/other/x'],
		['https://mojedane.gov.cz/dpr'],
		['/dpr/x'],
		['']
	])('rejects %s', (raw) => {
		expect(isEPOFormUrl(raw)).toBe(false);
	});
});
