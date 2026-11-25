import { describe, it, expect } from 'vitest';
import { helpTopics, getHelpTopics, type HelpTopicId } from './help-content';
import type { TaxConstants } from '$lib/api/client';

describe('help-content', () => {
	const topicIds = Object.keys(helpTopics) as HelpTopicId[];

	it('has all expected topics', () => {
		expect(topicIds.length).toBe(84);
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
			'frekvence-opakovani',
			'vystupni-dph',
			'vstupni-dph',
			'preneseni-danove-povinnosti',
			'nadmerny-odpocet',
			'zaklad-dane',
			'sekce-kontrolni-hlaseni',
			'dppd',
			'kod-plneni',
			'zdanovaci-obdobi',
			'typ-faktury',
			'dobropis',
			'vyrovnani-zalohy',
			'isdoc-export',
			'danova-kontrola',
			'ocr-import',
			'platebni-podminky',
			'email-sablony',
			'opakovane-faktury',
			'kategorie-nakladu',
			'duplikace-faktury',
			'rocni-dane',
			'pausalni-vydaje',
			'dan-15-23',
			'vymerovaci-zaklad',
			'casovy-test',
			'sleva-na-poplatnika',
			'zvyhodneni-na-deti',
			'mesice-proporcializace',
			'nezdanitelne-odpocty',
			'prehled-cssz',
			'prehled-zp',
			'kapitalove-prijmy-s8',
			'obchody-cp-s10',
			'nutno-priznat-dp',
			'doplatek-preplatek',
			'srazena-dan',
			'kurz-cnb',
			'nova-zaloha',
			'ztpp',
			'fifo-prepocet',
			'sleva-na-manzela',
			'zavisla-cinnost-s6',
			'dpc-dpp-hpp',
			'potvrzeni-zalohove',
			'potvrzeni-srazkove',
			'srazkova-do-dap',
			'radek-31-prijmy-s6',
			'radek-33-zahranicni-dan',
			'radek-34-dilci-zaklad-s6',
			'radek-84-srazene-zalohy',
			'radek-87-srazena-dan',
			'radek-89-vyplacene-bonusy',
			'rocni-zuctovani-rz',
			'superhruba-mzda-zrusena',
			'progresivni-sazba-23',
			'samostatny-zaklad-16a'
		];
		expect(topicIds.sort()).toEqual(expectedIds.sort());
	});

	describe('getHelpTopics with TaxConstants', () => {
		const mockConstants: TaxConstants = {
			rule_set_id: 'cz-dpfo-2024.v2',
			rule_set_status: 'final',
			rule_set_hash: 'abc123',
			year: 2024,
			basic_credit: 30840,
			spouse_credit: 24840,
			spouse_income_limit: 68000,
			student_credit: 4020,
			disability_credit_1: 2520,
			disability_credit_3: 5040,
			disability_ztpp: 16140,
			child_benefit_1: 15204,
			child_benefit_2: 22320,
			child_benefit_3_plus: 27840,
			progressive_threshold: 1582812,
			flat_rate_caps: { '80': 1600000, '60': 1200000, '40': 800000, '30': 600000 },
			deduction_cap_mortgage: 150000,
			deduction_cap_pension: 0,
			deduction_cap_life_insurance: 0,
			deduction_cap_savings_combined: 48000,
			deduction_cap_union: 3000,
			time_test_years: 3,
			security_exemption_limit: 100000
		};

		it('interpolates year-specific amounts into dynamic topics', () => {
			const topics = getHelpTopics(mockConstants);
			expect(topics['sleva-na-poplatnika'].simple).toContain('30');
			expect(topics['sleva-na-poplatnika'].simple).toContain('2024');
			expect(topics['zvyhodneni-na-deti'].simple).toContain('15');
			expect(topics['zvyhodneni-na-deti'].simple).toContain('2024');
			expect(topics['dan-15-23'].simple).toContain('2024');
			expect(topics['sleva-na-manzela'].simple).toContain('68');
			expect(topics['nezdanitelne-odpocty'].simple).toContain('150');
			expect(topics['ztpp'].simple).toContain('24');
			expect(topics['pausalni-vydaje'].simple).toContain('2024');
		});

		it('states that the child tax bonus has no annual cap', () => {
			const topic = getHelpTopics(mockConstants)['zvyhodneni-na-deti'];
			expect(topic.simple).toContain('bez ročního stropu');
			expect(topic.legal).toContain('zrušen');
		});

		it('returns generic text without constants', () => {
			const topics = getHelpTopics(null);
			expect(topics['sleva-na-poplatnika'].simple).not.toContain('2024');
			expect(topics['sleva-na-poplatnika'].simple).toContain('zdaňovacím období');
			expect(topics['zvyhodneni-na-deti'].simple).not.toContain('15 204');
			expect(topics['dan-15-23'].simple).toContain('36násobku');
		});

		it('describes the statutory advance and its first month', () => {
			const topics = getHelpTopics(null);
			const topic = topics['nova-zaloha'];

			expect(topic.legal).toContain('počtem měsíců výkonu činnosti');
			expect(topic.legal).toContain('§ 8 odst. 2');
			expect(topic.legal).not.toContain('§ 7 odst. 2');
			expect(topic.legal).toContain('§ 14 odst. 1, 2, 5, 6 a 13');
			expect(topic.legal).toContain('§ 2 odst. 1');
			expect(topic.legal).toContain('§ 3a odst. 2');
			expect(topic.legal).not.toContain('1/12');
			expect(topic.legal).not.toContain('50 % z minimálního vyměřovacího základu');
			expect(topic.simple).toContain('29,2 %');
			expect(topic.simple).toContain('13,5 %');
			expect(topic.simple).toContain('od měsíce následujícího po měsíci, kdy jste přehled podali');
			expect(topic.simple).toContain('už od měsíce, kdy jste přehled podali');
			expect(topic.simple).not.toContain('zatím nekontroluje');
			expect(topic.simple).toContain('Aplikace zálohu na toto minimum zvýší');
			expect(topics['prehled-cssz'].legal).toContain('nebo měl být podán');
			expect(topics['prehled-cssz'].legal).toContain('§ 14a odst. 3');
			expect(topics['prehled-cssz'].simple).toContain('sami si v něm vypočítáte');
			expect(topics['prehled-zp'].legal).toContain('§ 8 odst. 5');
			expect(topics['prehled-zp'].simple).toContain('sami si v něm vypočítáte');
		});

		it('returns same topics with and without constants', () => {
			const withConstants = getHelpTopics(mockConstants);
			const withoutConstants = getHelpTopics(null);
			const keysA = Object.keys(withConstants).sort();
			const keysB = Object.keys(withoutConstants).sort();
			expect(keysA).toEqual(keysB);
		});

		it('no help entry cites the repealed flat-rate restriction on credits (§ 35ca)', () => {
			const allTopics = [helpTopics, getHelpTopics(null), getHelpTopics(mockConstants)];
			for (const topics of allTopics) {
				for (const id of Object.keys(topics) as HelpTopicId[]) {
					for (const text of [topics[id].title, topics[id].simple, topics[id].legal]) {
						expect(text).not.toContain('§ 35ca');
						expect(text.toLowerCase()).not.toContain('nelze uplatnit slevu na manžela');
					}
				}
			}
		});
	});
});
