import { describe, it, expect } from 'vitest';
import { helpTopics, type HelpTopicId } from './help-content';

describe('help-content', () => {
	const topicIds = Object.keys(helpTopics) as HelpTopicId[];

	it('has all expected topics', () => {
		expect(topicIds.length).toBe(28);
	});

	it.each(topicIds)('topic "%s" has non-empty title', (id) => {
		expect(helpTopics[id].title).toBeTruthy();
		expect(helpTopics[id].title.length).toBeGreaterThan(0);
	});

	it.each(topicIds)('topic "%s" has non-empty simple explanation', (id) => {
		expect(helpTopics[id].simple).toBeTruthy();
		expect(helpTopics[id].simple.length).toBeGreaterThan(10);
	});

	it.each(topicIds)('topic "%s" has non-empty legal text', (id) => {
		expect(helpTopics[id].legal).toBeTruthy();
		expect(helpTopics[id].legal.length).toBeGreaterThan(10);
	});

	it('all topic IDs match the HelpTopicId type', () => {
		const expectedIds: HelpTopicId[] = [
			'variabilni-symbol',
			'konstantni-symbol',
			'duzp',
			'datum-splatnosti',
			'zpusob-platby',
			'poznamka-faktura',
			'poznamka-interni',
			'qr-platba',
			'danove-uznatelny',
			'podil-podnikani',
			'sazba-dph',
			'cislo-dokladu',
			'ico',
			'dic',
			'ares',
			'iban',
			'swift-bic',
			'platce-dph',
			'priznani-dph',
			'kontrolni-hlaseni',
			'souhrnne-hlaseni',
			'typ-podani',
			'ciselne-rady',
			'prefix-format',
			'prijmy-naklady',
			'neuhrazene-faktury',
			'faktury-po-splatnosti',
			'frekvence-opakovani'
		];
		expect(topicIds.sort()).toEqual(expectedIds.sort());
	});
});
