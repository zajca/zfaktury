export type HelpTopicId =
	| 'variabilni-symbol'
	| 'konstantni-symbol'
	| 'duzp'
	| 'datum-splatnosti'
	| 'zpusob-platby'
	| 'poznamka-faktura'
	| 'poznamka-interni'
	| 'qr-platba'
	| 'danove-uznatelny'
	| 'podil-podnikani'
	| 'sazba-dph'
	| 'cislo-dokladu'
	| 'ico'
	| 'dic'
	| 'ares'
	| 'iban'
	| 'swift-bic'
	| 'platce-dph'
	| 'priznani-dph'
	| 'kontrolni-hlaseni'
	| 'souhrnne-hlaseni'
	| 'typ-podani'
	| 'ciselne-rady'
	| 'prefix-format'
	| 'prijmy-naklady'
	| 'neuhrazene-faktury'
	| 'faktury-po-splatnosti'
	| 'frekvence-opakovani'
	| 'vystupni-dph'
	| 'vstupni-dph'
	| 'preneseni-danove-povinnosti'
	| 'nadmerny-odpocet'
	| 'zaklad-dane'
	| 'sekce-kontrolni-hlaseni'
	| 'dppd'
	| 'kod-plneni'
	| 'zdanovaci-obdobi'
	| 'typ-faktury'
	| 'dobropis'
	| 'vyrovnani-zalohy'
	| 'isdoc-export'
	| 'danova-kontrola'
	| 'ocr-import'
	| 'platebni-podminky'
	| 'email-sablony'
	| 'opakovane-faktury'
	| 'kategorie-nakladu'
	| 'duplikace-faktury'
	| 'rocni-dane'
	| 'pausalni-vydaje'
	| 'dan-15-23'
	| 'vymerovaci-zaklad'
	| 'casovy-test'
	| 'sleva-na-poplatnika'
	| 'zvyhodneni-na-deti'
	| 'mesice-proporcializace'
	| 'nezdanitelne-odpocty'
	| 'prehled-cssz'
	| 'prehled-zp'
	| 'kapitalove-prijmy-s8'
	| 'obchody-cp-s10'
	| 'nutno-priznat-dp'
	| 'doplatek-preplatek'
	| 'srazena-dan'
	| 'kurz-cnb'
	| 'nova-zaloha'
	| 'ztpp'
	| 'fifo-prepocet'
	| 'sleva-na-manzela';

export interface HelpTopic {
	title: string;
	simple: string;
	legal: string;
}

import type { TaxConstants } from '$lib/api/client';

function fmtCZK(n: number): string {
	return n.toLocaleString('cs-CZ') + ' Kč';
}

// Static topics that never change based on year.
const staticTopics: Record<string, HelpTopic> = {
	'variabilni-symbol': {
		title: 'Variabilní symbol',
		simple:
			'Variabilní symbol je číslo, které identifikuje platbu. Když vám někdo pošle peníze na účet, banka podle variabilního symbolu pozná, ke které faktuře platba patří.\n\nVětšinou se používá číslo faktury nebo jeho část. Důležité je, aby každá faktura měla unikátní variabilní symbol -- jinak nepoznáte, kdo za co platil.',
		legal:
			'Variabilní symbol je numerické pole o maximální délce 10 číslic. Je definován vyhláškou ČNB č. 169/2011 Sb. jako identifikátor transakce v tuzemském platebním styku.\n\nPodle zákona č. 284/2009 Sb. o platebním styku je variabilní symbol součást platebního příkazu a slouží k identifikaci platby mezi plátcem a příjemcem. Není povinný ze zákona, ale je standardní součástí fakturační praxe v ČR.'
	},
	'konstantni-symbol': {
		title: 'Konstantní symbol',
		simple:
			'Konstantní symbol je číslo, které říká, o jaký typ platby se jedná (např. platba za zboží, služby, nájem). V praxi se dnes používá minimálně -- většina bank ho nevyžaduje a pro OSVČ není potřebný.\n\nPokud si nejste jisti, můžete pole nechat prázdné.',
		legal:
			'Konstantní symbol je definován vyhláškou ČNB č. 169/2011 Sb. Jedná se o čtyřčíselný kód charakterizující platbu z hlediska jejího účelu. Od roku 2004 není jeho uvádění povinné pro běžné platby.\n\nNejčastější hodnoty: 0008 (platba za zboží), 0308 (platba za služby), 0558 (ostatní bezhotovostní platby).'
	},
	duzp: {
		title: 'Datum uskutečnění zdanitelného plnění (DUZP)',
		simple:
			'DUZP je datum, kdy skutečně došlo k dodání zboží nebo poskytnutí služby. Ne kdy jste vystavili fakturu, ne kdy vám přišly peníze -- ale kdy jste reálně odvedli práci nebo dodali produkt.\n\nNapř. pokud jste programovali web celý leden a fakturujete až 5. února, DUZP bude poslední den, kdy jste práci předali (třeba 31. ledna).\n\nPro plátce DPH je DUZP klíčové, protože určuje, do kterého zdaňovacího období faktura patří.',
		legal:
			'DUZP je definováno v zákoně č. 235/2004 Sb. o DPH, § 21. U dodání zboží je to den dodání (§ 21 odst. 1). U poskytování služeb den poskytnutí nebo den vystavení daňového dokladu, pokud nastal dříve (§ 21 odst. 3).\n\nPlátce DPH je povinen přiznat daň na výstupu ke dni uskutečnění zdanitelného plnění (§ 20a). DUZP určuje zdaňovací období, ve kterém musí být daň odvedena.'
	},
	'datum-splatnosti': {
		title: 'Datum splatnosti',
		simple:
			'Datum splatnosti je den, do kterého má odběratel zaplatit fakturu. Pokud zákazník nezaplatí do tohoto data, faktura je "po splatnosti" a můžete uplatňovat úroky z prodlení.\n\nBěžná splatnost je 14 nebo 30 dní od data vystavení. Může být i delší -- záleží na dohodě s odběratelem.',
		legal:
			'Splatnost je smluvní ujednání dle zákona č. 89/2012 Sb. (občanský zákoník), § 1958-1964. Pokud není dohodnuta, je splatnost bez zbytečného odkladu po doručení faktury.\n\nPodle zákona č. 340/2015 Sb. o registru smluv a § 1963 občanského zákoníku platí pro vztahy s veřejným sektorem maximální splatnost 30 dní. Pro obchodní vztahy mezi podnikateli je smluvní splatnost maximálně 60 dní (§ 1963a OZ), pokud to není vůči věřiteli hrubě nespravedlivé.'
	},
	'zpusob-platby': {
		title: 'Způsob platby',
		simple:
			'Způsob platby určuje, jak odběratel zaplatí fakturu. Nejčastěji bankovním převodem -- v tom případě faktura obsahuje číslo účtu a variabilní symbol.\n\nDalší možnosti jsou hotovost, platba kartou nebo dobírka. Pro účetní a daňové účely je důležité, aby způsob platby odpovídal realitě.',
		legal:
			'Způsob platby na faktuře není povinnou náležitostí daňového dokladu dle § 29 zákona č. 235/2004 Sb. o DPH. Jedná se však o běžnou obchodní náležitost.\n\nPro hotovostní platby platí limit 270 000 Kč dle zákona č. 254/2004 Sb. o omezení plateb v hotovosti (§ 4). Porušení je správní delikt s pokutou do 500 000 Kč pro fyzické osoby.'
	},
	'poznamka-faktura': {
		title: 'Poznámka na faktuře',
		simple:
			'Text, který se zobrazí přímo na faktuře, kterou pošlete zákazníkovi. Můžete sem napsat např. poděkování za spolupráci, informaci o probíhající akci nebo upozornění na změnu bankovního účtu.\n\nTato poznámka je viditelná pro odběratele.',
		legal:
			'Poznámka na faktuře není povinnou náležitostí daňového dokladu dle § 29 zákona č. 235/2004 Sb. Pokud však slouží jako informace o osvobozeném plnění, musí obsahovat odkaz na příslušné ustanovení zákona (§ 29 odst. 2 písm. c).\n\nNapř. u osvobozených plnění: "Osvobozeno od DPH dle § 51 zákona č. 235/2004 Sb."'
	},
	'poznamka-interni': {
		title: 'Interní poznámka',
		simple:
			'Soukromá poznámka, kterou vidíte jen vy. Na faktuře se nezobrazuje. Můžete sem napsat cokoliv pro vlastní evidenci -- např. "dohodnuto s Petrem 15.3.", "sleva za doporučení" apod.',
		legal: 'Interní poznámka nemá právní relevanci a neobjevuje se na žádném dokladu. Slouží pouze pro interní evidenci podnikatele.'
	},
	'qr-platba': {
		title: 'QR platba',
		simple:
			'QR kód na faktuře umožní odběrateli naskenovat platbu mobilem. Po naskenování se v bankovní aplikaci automaticky předvyplní číslo účtu, částka, variabilní symbol a další údaje.\n\nOdběratel tak nemusí nic opisovat a platba proběhne bez chyb. QR platba je standard České bankovní asociace.',
		legal:
			'QR platba (SPD -- Short Payment Descriptor) je standard České bankovní asociace pro mobilní platby. Formát je definován specifikací CBA a je podporován všemi hlavními bankami v ČR.\n\nFormát QR kódu: SPD*1.0*ACC:{IBAN}*AM:{částka}*CC:CZK*X-VS:{variabilní symbol}*...'
	},
	'danove-uznatelny': {
		title: 'Daňově uznatelný náklad',
		simple:
			'Daňově uznatelný náklad je výdaj, který si můžete odečíst od příjmů a tím snížit daň z příjmů. Musí splňovat podmínku: byl vynaložen na dosažení, zajištění a udržení vašich příjmů.\n\nPříklad: Notebook pro práci = daňově uznatelný. Dovolená = není daňově uznatelná.\n\nPokud používáte skutečné výdaje (ne paušální), je důležité správně označit, které náklady jsou daňově uznatelné.',
		legal:
			'Daňově uznatelné náklady jsou definovány v § 24 zákona č. 586/1992 Sb. o daních z příjmů. Jsou to výdaje vynaložené na dosažení, zajištění a udržení zdanitelných příjmů.\n\nDaňově neuznatelné náklady vyčte § 25 téhož zákona (např. repre, pokuty, penále). Prokazují se daňovými doklady -- podnikatel musí být schopen prokázat účetní doklad, účel výdaje a souvislost s podnikatelskou činností.'
	},
	'podil-podnikani': {
		title: 'Podíl pro podnikání',
		simple:
			'Některé náklady používáte jak pro podnikání, tak pro osobní účely. Např. auto, telefon nebo internet. Podíl pro podnikání určuje, kolik procent nákladů uplatníte jako daňový výdaj.\n\nPříklad: Telefon používáte z 60 % pro práci a z 40 % soukromě. Podíl pro podnikání je 60 % a jako daňový náklad si uplatníte 60 % z ceny.\n\nPoměrně je třeba rozdělit i DPH na vstupu, pokud jste plátce DPH.',
		legal:
			'Krácení nákladů u majetku používaného i pro soukromé účely upravuje § 24 odst. 2 písm. h) zákona č. 586/1992 Sb. Náklady se uplatňují v poměrné výši odpovídající rozsahu použití pro podnikatelskou činnost.\n\nPro DPH platí nárok na odpočet v poměrné výši dle § 75 zákona č. 235/2004 Sb. Podnikatel je povinen vést evidenci použití majetku pro podnikatelské a soukromé účely.'
	},
	'sazba-dph': {
		title: 'Sazba DPH',
		simple:
			'Sazba DPH (daň z přidané hodnoty) určuje, kolik procent daně se přidá k ceně zboží či služby. V Česku jsou aktuálně dvě sazby:\n\n- 21 % -- základní sazba (většina zboží a služeb)\n- 12 % -- snížená sazba (potraviny, léky, knihy, ubytování, stavební práce)\n\nPokud nejste plátce DPH, DPH neúčtujete a na faktuře uvedete 0 %.',
		legal:
			'Sazby DPH stanovuje § 47 zákona č. 235/2004 Sb. o DPH. Od 1. 1. 2024 platí dvě sazby: základní 21 % a snížená 12 % (sloučení původních dvou snížených sazeb 15 % a 10 %).\n\nZařazení zboží a služeb do snížené sazby je v příloze č. 2, 3 a 3a téhož zákona. Neplátce DPH není oprávněn vyúčtovat daň a nesmí ji uvést na dokladu (§ 26 odst. 3).'
	},
	'cislo-dokladu': {
		title: 'Číslo dokladu',
		simple:
			'Číslo dokladu jednoznačně identifikuje účetní doklad (fakturu, účtenku, pokladní doklad). Slouží pro evidenci -- abyste každý náklad snadno dohledali.\n\nMůže to být číslo z přijaté faktury od dodavatele, nebo vaše vlastní číslo, pokud doklad nemáte (např. pokladní blok označíte "P-001").',
		legal:
			'Pořadové číslo dokladu je povinnou náležitostí daňového dokladu dle § 29 odst. 1 písm. b) zákona č. 235/2004 Sb. Musí být přiřazeno v rámci jedné či více číselných řad, které zaručují jeho jednoznačnost.\n\nI pro neplátce DPH je jednoznačná identifikace dokladu povinností dle § 11 zákona č. 563/1991 Sb. o účetnictví.'
	},
	ico: {
		title: 'Identifikační číslo osoby (IČO)',
		simple:
			'IČO je osmimístné číslo, které dostane každý podnikatel nebo firma při registraci. Slouží k jednoznačné identifikaci -- jako "rodné číslo" pro podnikání.\n\nIČO se uvádí na všech fakturách a obchodních dokumentech. Podle IČO si můžete ověřit odběratele v obchodním rejstříku nebo registru ARES.',
		legal:
			'IČO je definováno zákonem č. 111/2009 Sb. o základních registrech, § 24-26. Přiděluje ho registrační orgán (živnostenský úřad, rejstříkový soud).\n\nPodle § 29 odst. 1 písm. a) zákona č. 235/2004 Sb. je IČO povinnou náležitostí daňového dokladu. Povinnost uvádět IČO na obchodních listinách plyne také z § 435 zákona č. 89/2012 Sb. (občanský zákoník).'
	},
	dic: {
		title: 'Daňové identifikační číslo (DIČ)',
		simple:
			'DIČ je číslo, které identifikuje plátce daně. V Česku má formát "CZ" + IČO (např. CZ12345678). DIČ dostanete po registraci k DPH u finančního úřadu.\n\nPokud nejste plátce DPH, DIČ nemusíte uvádět. Pokud jste plátce, je DIČ povinné na každé faktuře.',
		legal:
			'DIČ je definováno v § 130 zákona č. 280/2009 Sb. (daňový řád). Pro účely DPH je upraveno v § 4a zákona č. 235/2004 Sb. -- u fyzických osob má formát "CZ" + rodné číslo, u právnických osob "CZ" + IČO.\n\nDIČ je povinnou náležitostí daňového dokladu dle § 29 odst. 1 písm. a) zákona č. 235/2004 Sb. pro plátce DPH.'
	},
	ares: {
		title: 'ARES -- Administrativní registr ekonomických subjektů',
		simple:
			'ARES je veřejný registr, kde si můžete ověřit údaje o jakémkoli podnikateli nebo firmě v Česku. Stačí zadat IČO a zjistíte název, sídlo, právní formu a další informace.\n\nV ZFaktury se ARES používá pro automatické doplnění údajů o odběrateli -- zadejte IČO a systém stáhne jméno a adresu automaticky.',
		legal:
			'ARES je informační systém veřejné správy provozovaný Ministerstvem financí ČR. Agreguje data z více registrů: obchodního rejstříku, živnostenského rejstříku, registru DPH a dalších.\n\nPřístup k datům je bezúplatný a veřejný dle zákona č. 106/1999 Sb. o svobodném přístupu k informacím. API ARES je dostupné na ares.gov.cz.'
	},
	iban: {
		title: 'IBAN -- Mezinárodní číslo bankovního účtu',
		simple:
			'IBAN je mezinárodní formát čísla bankovního účtu. V Česku začíná "CZ" a má celkem 24 znaků (např. CZ65 0800 0000 1920 0014 5399).\n\nIBAN se používá pro zahraniční platby, ale stále častěji i pro tuzemské. Na faktuře ho uvádějte, pokud máte zahraniční odběratele nebo pokud chcete uživateli usnadnit platbu QR kódem.',
		legal:
			'IBAN je standardizován normou ISO 13616. V ČR je povinný pro přeshraniční platby v rámci EU/EHP dle nařízení EP a Rady (EU) č. 260/2012 (SEPA nařízení).\n\nPro tuzemské platby IBAN povinný není, ale banky ho podporují a je součástí QR platebního formátu CBA. Český IBAN má formát: CZ + 2 kontrolní číslice + 4 číslice kód banky + 16 číslic číslo účtu.'
	},
	'swift-bic': {
		title: 'SWIFT/BIC kód',
		simple:
			'SWIFT kód (také BIC) identifikuje banku při mezinárodních platbách. Je to 8 nebo 11 znaků dlouhý kód (např. KOMBCZPP pro Komerční banku).\n\nUvádějte ho na fakturách pro zahraniční odběratele -- bez SWIFT kódu nemůže platba ze zahraničí dojít na správnou banku.',
		legal:
			'SWIFT (Society for Worldwide Interbank Financial Telecommunication) kód, formálně BIC (Bank Identifier Code), je standardizován normou ISO 9362.\n\nPro platby v rámci SEPA (Single Euro Payments Area) není BIC povinný od 1. 2. 2016 dle nařízení (EU) č. 260/2012. Pro platby mimo SEPA je BIC stále nutný pro správné směrování platby.'
	},
	'platce-dph': {
		title: 'Plátce DPH',
		simple:
			'Plátce DPH je podnikatel registrovaný k dani z přidané hodnoty. Musí k cenám svých služeb a zboží přičítovat DPH a odvádět ho státu. Na druhou stranu si může odpočíst DPH z nákupů souvisejících s podnikáním.\n\nPovinně se plátcem DPH stáváte, když váš obrat za 12 po sobě jdoucích měsíců překročí 2 miliony Kč. Může se stát i dobrovolně.',
		legal:
			'Registrace plátce DPH je upravena v § 6-6f zákona č. 235/2004 Sb. Povinnou registraci vyvolá překročení obratu 2 000 000 Kč za 12 po sobě jdoucích kalendářních měsíců (§ 6 odst. 1) -- platnost od 1. 1. 2025.\n\nPlátce je povinen podávat daňové přiznání (§ 101), kontrolní hlášení (§ 101c) a v některých případech souhrnné hlášení (§ 102). Daň se odvádí měsíčně nebo čtvrtletně dle § 99-99a.'
	},
	'priznani-dph': {
		title: 'Přiznání k DPH',
		simple:
			'Přiznání k DPH je formulář, který plátce DPH odevzdává finančnímu úřadu. Obsahuje přehled vaší daně na výstupu (DPH z vašich faktur) a daně na vstupu (DPH z vašich nákupů). Rozdíl buď zaplatíte státu, nebo vám stát vrátí.\n\nPodává se měsíčně nebo čtvrtletně, vždy do 25. dne následujícího měsíce.',
		legal:
			'Daňové přiznání k DPH upravuje § 101 zákona č. 235/2004 Sb. Plátce je povinen podat přiznání do 25 dnů po skončení zdaňovacího období (§ 101 odst. 1).\n\nZdaňovací období je kalendářní měsíc nebo čtvrtletí (§ 99-99a). Přiznání se podává elektronicky ve formátu XML na portál finanční správy (EPO). Vzor formuláře stanoví Ministerstvo financí vyhláškou.'
	},
	'kontrolni-hlaseni': {
		title: 'Kontrolní hlášení',
		simple:
			'Kontrolní hlášení je měsíční report pro finanční úřad, který obsahuje rozpis všech vašich faktur (vydaných i přijatých) s DPH. Slouží státu ke křížové kontrole -- ověřuje, že DPH, které vy účtujete na výstupu, si váš odběratel uplatnil na vstupu, a naopak.\n\nPodává se vždy do 25. dne následujícího měsíce. Fyzické osoby mohou podávat čtvrtletně.',
		legal:
			'Kontrolní hlášení je upraveno v § 101c-101i zákona č. 235/2004 Sb. Podává se elektronicky ve formátu XML.\n\nLhůty: právnické osoby měsíčně, fyzické osoby ve lhůtě pro podání daňového přiznání (§ 101e). Za nepodání hrozí pokuta 10 000-50 000 Kč (§ 101h). Za nepodání na výzvu až 500 000 Kč.\n\nObsahuje údaje o přijatých a uskutečněných plněních nad 10 000 Kč včetně DPH s identifikací obchodního partnera (DIČ).'
	},
	'souhrnne-hlaseni': {
		title: 'Souhrnné hlášení',
		simple:
			'Souhrnné hlášení podáváte, pokud dodáváte zboží nebo služby do jiných zemí EU plátcům DPH. Hlášení informuje finanční úřad o těchto dodávkách.\n\nPokud obchodujete pouze v Česku, souhrnné hlášení vás nezajímá.',
		legal:
			'Souhrnné hlášení upravuje § 102 zákona č. 235/2004 Sb. Podává se za každý kalendářní měsíc (při dodání zboží) nebo čtvrtletí (při poskytování služeb) do 25 dnů po skončení období.\n\nTýká se dodání zboží do jiného členského státu osobě registrované k DPH (§ 102 odst. 1 písm. a), poskytování služeb s místem plnění v jiném členském státě (§ 102 odst. 1 písm. d) a přemístění obchodního majetku (§ 102 odst. 1 písm. b).'
	},
	'typ-podani': {
		title: 'Typ podání',
		simple:
			'Typ podání určuje, zda se jedná o řádné, opravné nebo dodatečné podání:\n\n- Řádné -- první podání za dané období\n- Opravné -- oprava podání před uplynutím lhůty (nahradí původní)\n- Dodatečné -- oprava po uplynutí lhůty (podává se navíc k řádnému)',
		legal:
			'Typy podání definuje zákon č. 280/2009 Sb. (daňový řád):\n\n- Řádné podání (§ 135) -- standardní podání v zákonném termínu\n- Opravné podání (§ 138) -- nahrazuje původní podání před uplynutím lhůty, poslední podané platí\n- Dodatečné podání (§ 141) -- podává se po uplynutí lhůty pro řádné podání, pokud podnikatel zjistí chybu. Lhůta pro podání: do konce měsíce následujícího po zjištění chyby'
	},
	'ciselne-rady': {
		title: 'Číselné řady',
		simple:
			'Číselné řady zajišťují automatické číslování vašich faktur. Místo ručního zadávání čísel systém sám přiřadí další číslo v pořadí.\n\nMůžete mít více číselných řad -- např. jednu pro tuzemské faktury (FV-2024-001) a jinou pro zahraniční (ZF-2024-001).',
		legal:
			'Povinnost číselných řad vyplývá z § 29 odst. 1 písm. b) zákona č. 235/2004 Sb. -- daňový doklad musí obsahovat pořadové číslo přiřazené v rámci jedné či více číselných řad.\n\nČíselná řada musí zaručovat jednoznačnost dokladu. Podnikatel je povinen vést evidenci vydaných dokladů a jejich číselných řad pro účely případné kontroly finančním úřadem.'
	},
	'prefix-format': {
		title: 'Prefix a formát číselné řady',
		simple:
			'Prefix je text před číslem faktury (např. "FV" pro fakturu vydanou). Formát určuje, jak bude číslo vypadat -- např. "{prefix}{year}-{number:4}" vytvoří čísla jako FV2024-0001, FV2024-0002 atd.\n\nČíslování se resetuje na začátku každého roku, takže první faktura nového roku bude vždy 0001.',
		legal:
			'Formát číselné řady není zákonem předepsán. Zákon č. 235/2004 Sb. v § 29 vyžaduje pouze to, aby pořadové číslo bylo jednoznačné v rámci číselné řady.\n\nDoporučuje se včetně roku (např. 2024-001) pro snazší orientaci a průkaznost při daňové kontrole. Prefix pomáhá rozlišit typ dokladu (faktury vydané, přijaté, dobropisy atd.).'
	},
	'prijmy-naklady': {
		title: 'Příjmy a náklady',
		simple:
			'Příjmy jsou peníze, které vám zákazníci zaplatili za vaše služby nebo zboží. Náklady jsou peníze, které jste utratili v souvislosti s podnikáním.\n\nRozdíl mezi příjmy a náklady je základem daně -- čím více nákladů (daňově uznatelných) máte, tím méně daně zaplatíte.',
		legal:
			'Příjmy z podnikání OSVČ jsou upraveny v § 7 zákona č. 586/1992 Sb. o daních z příjmů. Základ daně se stanoví jako rozdíl mezi příjmy a výdaji (§ 23).\n\nAlternativně může OSVČ uplatnit paušální výdaje (§ 7 odst. 7): 80 % u řemeslných živností, 60 % u ostatních živností, 40 % u příjmů z jiného podnikání. Paušální výdaje jsou omezeny částkou 1 600 000 / 1 200 000 / 800 000 Kč.'
	},
	'neuhrazene-faktury': {
		title: 'Neuhrazené faktury',
		simple:
			'Neuhrazené faktury jsou faktury, které jste vystavili, ale zákazník je ještě nezaplatil. Mohou být před splatností (zákazník má ještě čas) nebo po splatnosti (zákazník je v prodlení).\n\nJe důležité sledovat neuhrazené faktury a včas upomínat dlužníky. Po splatnosti mají úroky z prodlení.',
		legal:
			'Neuhrazené pohledávky po splatnosti lze daňově odepsat dle § 24 odst. 2 písm. y) zákona č. 586/1992 Sb. u pohledávek za dlužníkem v insolvenčním řízení.\n\nOpravné položky k pohledávkám upravuje zákon č. 593/1992 Sb. o rezervách: po 18 měsících po splatnosti až 50 %, po 30 měsících až 100 % (§ 8a). Úroky z prodlení se řídí § 1970 občanského zákoníku -- repo sazba ČNB + 8 p.b.'
	},
	'faktury-po-splatnosti': {
		title: 'Faktury po splatnosti',
		simple:
			'Faktura je po splatnosti, když zákazník nezaplatil do data splatnosti. Od tohoto okamžiku je v prodlení a vy můžete uplatnit úroky z prodlení.\n\nDoporučený postup: po 7 dnech první upomínka, po 14 dnech druhá upomínka, po 30 dnech předsoudní upomínka s výhružkou právními kroky.',
		legal:
			'Prodlení dlužníka upravuje § 1968-1975 zákona č. 89/2012 Sb. (občanský zákoník). Dlužník, který svůj dluh řádně a včas neplní, je v prodlení (§ 1968).\n\nVěřitel má právo na úroky z prodlení (§ 1970) ve výši repo sazby ČNB + 8 procentních bodů. U obchodních vztahů má věřitel také právo na minimální paušál 1 200 Kč za náklady spojené s uplatněním pohledávky (nařízení vlády č. 351/2013 Sb.).'
	},
	'frekvence-opakovani': {
		title: 'Frekvence opakování',
		simple:
			'Frekvence určuje, jak často se opakující faktura automaticky vytvoří. Např. měsíční frekvence znamená, že se faktura vytvoří jednou měsíčně.\n\nBěžné frekvence: měsíční (např. paušální služby, nájem), čtvrtletní (např. pravidelné konzultace), roční (např. licence, předplatné).',
		legal:
			'Opakující se plnění (trvalé plnění) je upraveno v § 21 odst. 8 zákona č. 235/2004 Sb. U opakovaného plnění se DUZP stanoví nejpozději posledním dnem zdaňovacího období.\n\nSmlouvy na opakované plnění (např. nájem, servisní smlouvy) se řídí ustanoveními o závazkovém právu v občanském zákoníku (§ 1724 a násl. zákona č. 89/2012 Sb.).'
	},
	'vystupni-dph': {
		title: 'Výstupní DPH',
		simple:
			'Výstupní DPH je daň, kterou účtujete svým zákazníkům na fakturách. Když vystavíte fakturu s DPH, tuto daň musíte odvést státu.\n\nNapř. fakturujete službu za 10 000 Kč + 21 % DPH = 12 100 Kč. Těch 2 100 Kč je výstupní DPH, které odvedete finančnímu úřadu.',
		legal:
			'Výstupní DPH (daň na výstupu) je definováno v § 4 odst. 1 písm. c) zákona č. 235/2004 Sb. o DPH. Plátce je povinen přiznat daň na výstupu ke dni uskutečnění zdanitelného plnění (§ 20a) nebo ke dni přijetí úhrady, pokud nastala dříve (§ 21).\n\nDaň na výstupu se uvádí v daňovém přiznání v řádcích 1-13 formuláře.'
	},
	'vstupni-dph': {
		title: 'Vstupní DPH',
		simple:
			'Vstupní DPH je daň, kterou jste zaplatili při svých nákupech. Tuto daň si můžete odečíst od výstupního DPH -- tím snížíte částku, kterou odvedete státu.\n\nNapř. koupíte notebook za 24 200 Kč (20 000 + 4 200 DPH). Těch 4 200 Kč je vstupní DPH, které si odečtete.',
		legal:
			'Nárok na odpočet daně na vstupu upravují § 72-73 zákona č. 235/2004 Sb. Plátce má nárok na odpočet daně u přijatých zdanitelných plnění, která použije pro uskutečnění své ekonomické činnosti (§ 72 odst. 1).\n\nPodmínkou odpočtu je držení daňového dokladu (§ 73 odst. 1). Nárok na odpočet lze uplatnit nejdříve za zdaňovací období, ve kterém jsou splněny podmínky (§ 73 odst. 3).'
	},
	'preneseni-danove-povinnosti': {
		title: 'Přenesení daňové povinnosti',
		simple:
			'Přenesení daňové povinnosti (reverse charge) znamená, že DPH neplatí dodavatel, ale odběratel. Dodavatel vystaví fakturu bez DPH a odběratel si DPH sám vypočítá a přizná.\n\nPoužívá se např. u stavebních prací, dodání šrotu a odpadu, nebo u obchodů mezi firmami v rámci EU.',
		legal:
			'Přenesení daňové povinnosti (režim reverse charge) upravuje § 92a zákona č. 235/2004 Sb. U tuzemských plnění se týká zboží a služeb uvedených v příloze č. 6 zákona (stavební práce, šrot, odpady aj.).\n\nPři přenesení daňové povinnosti je odběratel povinen daň přiznat a má nárok na odpočet (§ 92a odst. 1). Dodavatel uvede plnění v řádku 25 daňového přiznání.'
	},
	'nadmerny-odpocet': {
		title: 'Nadměrný odpočet / Daňová povinnost',
		simple:
			'Výsledek DPH přiznání je buď daňová povinnost, nebo nadměrný odpočet:\n\n- Daňová povinnost: výstupní DPH > vstupní DPH -- rozdíl zaplatíte státu\n- Nadměrný odpočet: vstupní DPH > výstupní DPH -- stát vám vrátí rozdíl\n\nNadměrný odpočet vzniká např. při velkých investicích (nákup stroje, rekonstrukce).',
		legal:
			'Nadměrný odpočet je definován v § 4 odst. 1 písm. d) zákona č. 235/2004 Sb. Vznikne-li nadměrný odpočet, vrátí ho správce daně plátci do 30 dnů od vyměření (§ 105 odst. 1).\n\nSprávce daně může před vrácením zahájit postup k odstranění pochybností (§ 89 daňového řádu), čímž se lhůta prodlouží. Nadměrný odpočet se přednostně použije na úhradu případných daňových nedoplatků (§ 105 odst. 2).'
	},
	'zaklad-dane': {
		title: 'Základ daně',
		simple:
			'Základ daně je částka bez DPH, ze které se DPH vypočítá. Např. pokud je cena služby 12 100 Kč včetně 21 % DPH, základ daně je 10 000 Kč a DPH 2 100 Kč.\n\nV DPH přiznání se základ daně uvádí ve sloupcích vedle vypočtené daně.',
		legal:
			'Základ daně je definován v § 36 zákona č. 235/2004 Sb. Základem daně je vše, co jako úhradu obdržel nebo má obdržet plátce za uskutečněné zdanitelné plnění od osoby, pro kterou plnění uskutečnil, nebo od třetí osoby (§ 36 odst. 1).\n\nZáklad daně zahrnuje i vedlejší výdaje (balení, přeprava, pojištění) dle § 36 odst. 3.'
	},
	'sekce-kontrolni-hlaseni': {
		title: 'Sekce kontrolního hlášení (A4/A5/B2/B3)',
		simple:
			'Kontrolní hlášení se dělí na sekce podle směru a velikosti plnění:\n\n- A4: Vydané faktury nad 10 000 Kč včetně DPH (s detailem o odběrateli)\n- A5: Vydané faktury do 10 000 Kč (souhrnně, bez detailu)\n- B2: Přijaté faktury nad 10 000 Kč včetně DPH (s detailem o dodavateli)\n- B3: Přijaté faktury do 10 000 Kč (souhrnně, bez detailu)\n\nU A4 a B2 se uvádí DIČ partnera, číslo dokladu a další údaje.',
		legal:
			'Členění kontrolního hlášení stanovuje § 101c-101d zákona č. 235/2004 Sb. a pokyn GFŘ-D-57.\n\nOddíl A obsahuje údaje o uskutečněných plněních (výstupy): A4 = plnění nad 10 000 Kč s identifikací odběratele, A5 = ostatní plnění. Oddíl B obsahuje údaje o přijatých plněních (vstupy): B2 = plnění nad 10 000 Kč s identifikací dodavatele, B3 = ostatní plnění.\n\nRozhodující částka 10 000 Kč je včetně DPH.'
	},
	dppd: {
		title: 'Datum poskytnutí daňového plnění (DPPD)',
		simple:
			'DPPD je datum, které se uvádí v kontrolním hlášení. Odpovídá datu uskutečnění plnění (DUZP) z faktury.\n\nPozor: DPPD není datum vystavení faktury ani datum splatnosti -- je to den, kdy skutečně došlo k dodání zboží nebo poskytnutí služby.',
		legal:
			'DPPD (datum poskytnutí/přijetí plnění) se uvádí v kontrolním hlášení dle § 101c zákona č. 235/2004 Sb. Odpovídá datu uskutečnění zdanitelného plnění (DUZP) dle § 21 téhož zákona.\n\nV oddílech A4 a B2 kontrolního hlášení se DPPD uvádí u každého řádku. V oddílech A5 a B3 se neuvádí (plnění jsou agregována).'
	},
	'kod-plneni': {
		title: 'Kód plnění',
		simple:
			'Kód plnění v souhrnném hlášení určuje typ obchodu s partnerem v EU:\n\n- 0: Dodání zboží do jiné členské země\n- 1: Poskytnutí služby podle § 9 odst. 1 (místo plnění u příjemce)\n- 2: Obchod v rámci triangulace (třetí strana)\n- 3: Poskytnutí služby podle § 54 (finanční a pojišťovací služby)',
		legal:
			'Kódy plnění jsou definovány v § 102 zákona č. 235/2004 Sb. a v pokynu GFŘ k vyplňování souhrnného hlášení.\n\nKód 0: dodání zboží osobě registrované k DPH v jiném členském státě (§ 102 odst. 1 písm. a). Kód 1: poskytnutí služby s místem plnění dle § 9 odst. 1 (§ 102 odst. 1 písm. d). Kód 2: dodání zboží v rámci zjednodušeného postupu při třístranném obchodu (§ 102 odst. 1 písm. c). Kód 3: poskytnutí služby dle § 54.'
	},
	'zdanovaci-obdobi': {
		title: 'Zdaňovací období',
		simple:
			'Zdaňovací období je časový úsek, za který podáváte DPH přiznání a odvádíte daň. Může být:\n\n- Měsíční: přiznání podáváte každý měsíc (povinně při obratu nad 10 mil. Kč)\n- Čtvrtletní: přiznání podáváte za každé čtvrtletí (pro menší plátce DPH)\n\nPřiznání se vždy podává do 25. dne po skončení období.',
		legal:
			'Zdaňovací období upravují § 99-99a zákona č. 235/2004 Sb. Základním zdaňovacím obdobím je kalendářní měsíc (§ 99). Plátce může zvolit čtvrtletní období, pokud jeho obrat za předcházející kalendářní rok nepřesáhl 10 000 000 Kč a není nespolehlivým plátcem (§ 99a).\n\nZměna zdaňovacího období se oznamuje správci daně do konce ledna příslušného roku (§ 99a odst. 2).'
	},
	'typ-faktury': {
		title: 'Typ dokladu',
		simple:
			'Faktura je daňový doklad, který vystavujete za dodané zboží nebo služby. Zálohová faktura (proforma) je výzva k platbě -- není daňovým dokladem a neslouží k uplatnění DPH.\n\nPokud jste plátce DPH, po úhradě zálohové faktury musíte vystavit řádnou fakturu (vyrovnání zálohy).',
		legal:
			'Daňový doklad je definován v § 26 zákona č. 235/2004 Sb. o DPH. Zálohová faktura není daňovým dokladem ve smyslu tohoto zákona -- jedná se o obchodní dokument vyzývající k platbě.\n\nPovinné náležitosti daňového dokladu upravuje § 29 téhož zákona. Po přijetí úhrady zálohové faktury vzniká povinnost vystavit řádný daňový doklad dle § 28 odst. 2.'
	},
	'dobropis': {
		title: 'Dobropis (opravný daňový doklad)',
		simple:
			'Dobropis je opravný doklad, který vystavujete, když potřebujete snížit částku na již vydané faktuře. Typické důvody: sleva, reklamace, chybně účtovaná částka nebo vrácení zboží.\n\nDobropis odkazuje na původní fakturu a obsahuje zápornou částku. Po jeho vystavení se sníží vaše daňové závazky.',
		legal:
			'Opravný daňový doklad upravuje § 42 zákona č. 235/2004 Sb. o DPH. Plátce je povinen vystavit opravný daňový doklad do 15 dnů ode dne zjištění skutečností rozhodných pro provedení opravy (§ 42 odst. 2).\n\nOpravný doklad musí obsahovat důvod opravy, rozdíl mezi původní a novou částkou a odkaz na původní daňový doklad (§ 45 odst. 1).'
	},
	'vyrovnani-zalohy': {
		title: 'Vyrovnání zálohy',
		simple:
			'Po zaplacení zálohové faktury (proformy) je třeba vystavit řádnou fakturu. Tato faktura obsahuje celkovou částku za dodané zboží či služby, od které se odečte již uhrazená záloha.\n\nVýsledkem je doplatek, který zákazník ještě uhradí, nebo nulová částka, pokud záloha pokryla vše.',
		legal:
			'Povinnost vystavit daňový doklad po přijetí úhrady vyplývá z § 21 odst. 1 zákona č. 235/2004 Sb. o DPH. Dnem přijetí úhrady vzniká povinnost přiznat daň na výstupu.\n\nPři vyrovnání se na řádné faktuře uvede celková částka plnění a odečte se dříve uhrazená záloha. Základ daně a DPH se vypočtou z celkové částky plnění.'
	},
	'isdoc-export': {
		title: 'Export ISDOC',
		simple:
			'ISDOC je český standard pro elektronickou fakturaci. Soubor ve formátu ISDOC (.isdoc) obsahuje všechna data faktury ve strojově čitelné podobě.\n\nKdyž pošlete fakturu ve formátu ISDOC, odběratelův účetní systém ji může automaticky načíst bez ručního přepisování.',
		legal:
			'ISDOC (Information System Document) je český národní standard elektronické fakturace definovaný ICT Unií. Formát je založený na UN/CEFACT a je kompatibilní s evropskou normou EN 16931.\n\nPoužívání elektronických faktur upravuje § 26 odst. 3 a § 34 zákona č. 235/2004 Sb. Elektronická faktura musí být opatřena zaručenými prostředky pro ověření původu a neporušenosti obsahu.'
	},
	'danova-kontrola': {
		title: 'Daňová kontrola nákladů',
		simple:
			'Daňová kontrola nákladů je proces, kdy systematicky projdete své výdaje a ověříte, že každý náklad je správně doložen, správně zařazen a daňově uznatelný.\n\nOznačením nákladu jako "zkontrolovaný" si udržujete přehled o tom, které výdaje jste již ověřili a které ještě čekají na kontrolu.',
		legal:
			'Daňově uznatelné náklady jsou definovány v § 24-25 zákona č. 586/1992 Sb. o daních z příjmů. Podnikatel je povinen prokázat, že výdaj byl vynaložen na dosažení, zajištění a udržení zdanitelných příjmů.\n\nSprávce daně může v rámci daňové kontroly (§ 85 zákona č. 280/2009 Sb.) požadovat prokázání oprávněnosti všech uplatněných nákladů. Pravidelná kontrola minimalizuje riziko doplacení daně.'
	},
	'ocr-import': {
		title: 'Import z dokladu (OCR)',
		simple:
			'OCR (optické rozpoznávání znaků) automaticky přečte text z nahrané faktury nebo účtenky. Stačí nahrát soubor (PDF, JPG, PNG nebo WebP) a systém se pokusí rozpoznat dodavatele, částku, datum a další údaje.\n\nRozpoznaná data můžete před uložením zkontrolovat a upravit.',
		legal:
			'Archivace daňových dokladů v elektronické podobě je upravena v § 35a zákona č. 235/2004 Sb. a § 31-32 zákona č. 563/1991 Sb. o účetnictví. Elektronická kopie musí zachovat věrnost a čitelnost původního dokladu.\n\nPovinnost uchovat daňové doklady je 10 let od konce zdaňovacího období (§ 35 zákona č. 235/2004 Sb.).'
	},
	'platebni-podminky': {
		title: 'Platební podmínky',
		simple:
			'Splatnost ve dnech určuje, kolik dní od vystavení faktury má zákazník na zaplacení. Tato hodnota se automaticky nastaví na nových fakturách pro tohoto zákazníka.\n\nBěžná splatnost je 14 nebo 30 dní. Pro stálé zákazníky můžete nastavit individuální splatnost.',
		legal:
			'Splatnost je smluvní ujednání dle § 1958-1964 zákona č. 89/2012 Sb. (občanský zákoník). Pro obchodní vztahy mezi podnikateli je maximální smluvní splatnost 60 dní dle § 1963a OZ.\n\nPro vztahy s veřejným sektorem platí maximální splatnost 30 dní (§ 1963 OZ). Delší splatnost je možná jen pokud to není vůči věřiteli hrubě nespravedlivé.'
	},
	'email-sablony': {
		title: 'Šablony emailů',
		simple:
			'Šablona emailu určuje předmět a text zprávy, která se odešle zákazníkovi spolu s fakturou. Použijte {invoice_number} a systém automaticky vloží číslo faktury.\n\nŠablonu nastavíte jednou a pak se použije pro všechny odeslané faktury. Před odesláním můžete text ještě upravit.',
		legal:
			'Odeslání faktury emailem je běžnou obchodní praxí. Elektronické doručení daňového dokladu je upraveno v § 34 zákona č. 235/2004 Sb. -- odběratel musí s elektronickým doručením souhlasit.\n\nElektronická faktura musí splňovat podmínky pro ověření původu a neporušenosti obsahu (§ 34 odst. 1).'
	},
	'opakovane-faktury': {
		title: 'Opakované faktury',
		simple:
			'Opakované faktury jsou šablony, ze kterých se automaticky generují nové faktury v pravidelných intervalech (měsíčně, čtvrtletně, ročně).\n\nHodí se pro paušální služby, nájem, předplatné nebo jakoukoli pravidelnou fakturaci. Šablona obsahuje zákazníka, položky a frekvenci -- systém pak sám vytvoří fakturu když přišel čas.',
		legal:
			'Opakované plnění je upraveno v § 21 odst. 8 zákona č. 235/2004 Sb. o DPH. U opakujícího se plnění se DUZP stanoví nejpozději posledním dnem zdaňovacího období.\n\nSmlouvy na opakované plnění (nájem, servisní smlouvy) se řídí ustanoveními o závazkovém právu v občanském zákoníku (§ 1724 a násl. zákona č. 89/2012 Sb.).'
	},
	'kategorie-nakladu': {
		title: 'Kategorie nákladů',
		simple:
			'Kategorie pomáhají třídit náklady podle typu (kancelář, cestovné, služby, materiál apod.). Dobře roztříděné náklady usnadňují přehled o výdajích, přípravu daňového přiznání a komunikaci s účetním.\n\nMůžete použít výchozí kategorie nebo si vytvořit vlastní.',
		legal:
			'Třídění nákladů podle kategorií není zákonem předepsáno, ale vyplývá z povinnosti vést účetnictví přehledně a průkazně (§ 8 zákona č. 563/1991 Sb.).\n\nPro účely daňového přiznání je vhodné členit náklady dle § 24 zákona č. 586/1992 Sb. (daňově uznatelné) a § 25 (neuznatelné), příp. dle povahy výdaje pro správné vyplnění příloh přiznání.'
	},
	'duplikace-faktury': {
		title: 'Duplikace faktury',
		simple:
			'Duplikace vytvoří novou fakturu jako kopii stávající. Zkopíruje se zákazník, položky, způsob platby a další nastavení. Nová faktura dostane nové číslo a aktuální datumy.\n\nHodí se, když vystavujete podobnou fakturu jako minule -- nemusíte vše vyplňovat znovu.',
		legal:
			'Duplikovaná faktura je nový, samostatný daňový doklad s vlastním pořadovým číslem dle § 29 zákona č. 235/2004 Sb. Jedná se o zcela nezávislý doklad, nikoliv o kopii původního.\n\nPořadové číslo musí být unikátní v rámci číselné řady (§ 29 odst. 1 písm. b).'
	},
	'rocni-dane': {
		title: 'Roční daně a přehledy OSVČ',
		simple:
			'Roční daňové přiznání (DPFO) a přehledy pro sociální (ČSSZ) a zdravotní pojišťovnu (ZP). Aplikace spočítá základ daně z faktur a nákladů, aplikuje sazby a slevy, a vygeneruje XML pro elektronické podání.',
		legal:
			'Daňové přiznání k dani z příjmů fyzických osob (§ 38g zákona č. 586/1992 Sb.). Přehled o příjmech a výdajích OSVČ pro ČSSZ (§ 15 zákona č. 589/1992 Sb.) a pro zdravotní pojišťovnu (§ 24 zákona č. 592/1992 Sb.).'
	},
	'vymerovaci-zaklad': {
		title: 'Vyměřovací základ pro pojistné',
		simple:
			'Vyměřovací základ je částka, ze které se počítá sociální a zdravotní pojistné. Pro OSVČ je to 50 % ze základu daně (příjmy minus výdaje).\n\nExistuje minimální vyměřovací základ -- i když máte nízký zisk, zaplatíte pojistné alespoň z minima. U sociálního pojištění je minimum dobrovolné (pokud je hlavní činnost), u zdravotního je povinné vždy.',
		legal:
			'Vyměřovací základ pro sociální pojištění OSVČ: 50 % základu daně (§ 5b zákona č. 589/1992 Sb.). Minimální vyměřovací základ: 25 % průměrné mzdy pro hlavní činnost. Pro zdravotní pojištění: 50 % základu daně (§ 3a zákona č. 592/1992 Sb.), minimální základ je 50 % průměrné mzdy (§ 3a odst. 2).'
	},
	'casovy-test': {
		title: 'Časový test 3 roky pro cenné papíry',
		simple:
			'Pokud vlastníte akcii, ETF nebo jiný cenný papír déle než 3 roky a pak ho prodáte, zisk z prodeje je osvobozený od daně. Tomu se říká "časový test".\n\nPříklad: Koupíte akcii v lednu 2022 a prodáte v únoru 2025 (déle než 3 roky) -- neplatíte žádnou daň ze zisku. Pokud prodáte dříve, zisk se musí danit v rámci § 10.',
		legal:
			'Osvobození příjmů z prodeje cenných papírů po časovém testu upravuje § 4 odst. 1 písm. w) zákona č. 586/1992 Sb. Doba držení musí překročit 3 roky. Od 2025 se časový test prodlužuje na 3 roky i pro kryptoměny (§ 4 odst. 1 písm. x). Pro fondy kolektivního investování platí rovněž 3 roky (§ 4 odst. 1 písm. w).'
	},
	'mesice-proporcializace': {
		title: 'Proporcionalizace slev podle měsíců',
		simple:
			'Některé slevy a zvýhodnění se počítají v poměrné výši podle počtu měsíců, po které podmínka platila. Např. pokud jste se oženili v červnu, slevu na manžela/ku uplatníte za 7 měsíců (červen-prosinec).\n\nStejně to funguje u dětí -- pokud se dítě narodilo v říjnu, zvýhodnění uplatníte za 3 měsíce. Rozhoduje stav na začátku měsíce.',
		legal:
			'Proporcionalizace slev je upravena v § 35ba odst. 3 a § 35c odst. 8 zákona č. 586/1992 Sb. Sleva na manžela/ku a daňové zvýhodnění na dítě se uplatňují v poměrné výši odpovídající počtu kalendářních měsíců, na jejichž počátku byly splněny podmínky pro uplatnění.'
	},
	'prehled-cssz': {
		title: 'Přehled OSVČ pro ČSSZ',
		simple:
			'Přehled pro Českou správu sociálního zabezpečení je roční formulář, ve kterém vykazujete své příjmy a výdaje z podnikání. ČSSZ z něj vypočítá vaše pojistné a novou výši měsíčních záloh.\n\nPřehled se podává do jednoho měsíce po lhůtě pro podání daňového přiznání. Pokud vám vyšel doplatek, musíte ho zaplatit do 8 dnů od podání přehledu.',
		legal:
			'Povinnost podat přehled vyplývá z § 15 zákona č. 589/1992 Sb. o pojistném na sociální zabezpečení. Lhůta: do jednoho měsíce po lhůtě pro podání daňového přiznání (§ 15 odst. 1). Doplatek pojistného je splatný do 8 dnů po podání přehledu (§ 14a odst. 2). Nová výše zálohy platí od měsíce následujícího po měsíci podání přehledu.'
	},
	'prehled-zp': {
		title: 'Přehled OSVČ pro zdravotní pojišťovnu',
		simple:
			'Přehled pro zdravotní pojišťovnu je roční formulář, ve kterém vykazujete své příjmy a výdaje. Pojišťovna z něj vypočítá vaše zdravotní pojistné a novou výši měsíčních záloh.\n\nPřehled se podává do jednoho měsíce po lhůtě pro podání daňového přiznání. Doplatek se platí do 8 dnů od podání.',
		legal:
			'Povinnost podat přehled upravuje § 24 zákona č. 592/1992 Sb. o pojistném na všeobecné zdravotní pojištění. Lhůta: do jednoho měsíce po lhůtě pro podání daňového přiznání (§ 24 odst. 2). Doplatek pojistného je splatný do 8 dnů po podání přehledu (§ 7 odst. 2). OSVČ přehled podává té pojišťovně, u které byla pojištěna k 1. lednu příslušného roku.'
	},
	'kapitalove-prijmy-s8': {
		title: 'Kapitálové příjmy (§8)',
		simple:
			'Kapitálové příjmy zahrnují dividendy, úroky z vkladů, kupony z dluhopisů a výplaty z fondů. Většina těchto příjmů je zdaněna srážkovou daní (15 %) přímo u zdroje -- banka nebo broker daň strhne automaticky.\n\nDo daňového přiznání (§8) uvádíte jen příjmy, které nebyly zdaněny srážkovou daní, nebo zahraniční dividendy, kde chcete uplatnit zápočet daně.',
		legal:
			'Kapitálové příjmy jsou definovány v § 8 zákona č. 586/1992 Sb. Srážková daň 15 % dle § 36 odst. 2 se uplatní u dividend, úroků a dalších příjmů z § 8. Zahraniční kapitálové příjmy se uvádějí v přiznání a případná zahraniční srážková daň se započte dle smlouvy o zamezení dvojího zdanění (§ 38f).'
	},
	'obchody-cp-s10': {
		title: 'Obchody s CP a kryptem (§10)',
		simple:
			'Zisky z prodeje cenných papírů (akcií, ETF, dluhopisů) a kryptoměn se daní v rámci § 10 jako "ostatní příjmy". Od příjmů z prodeje si odečtete nabývací cenu (pořadí FIFO) a poplatky.\n\nZdanitelný je pouze zisk, a to jen pokud nepřešlo 3 roky od nákupu (časový test). Pokud celkové ostatní příjmy za rok nepřesáhnou 100 000 Kč, můžou být také osvobozeny.',
		legal:
			'Příjmy z prodeje cenných papírů a kryptoměn upravuje § 10 odst. 1 písm. b) zákona č. 586/1992 Sb. Výdajem je nabývací cena dle § 10 odst. 4. Osvobození po časovém testu 3 roky dle § 4 odst. 1 písm. w). Limit osvobození pro ostatní příjmy do 100 000 Kč dle § 10 odst. 3 písm. a). Ztráta z § 10 se nekompenzuje se zisky z § 7.'
	},
	'nutno-priznat-dp': {
		title: 'Kdy přiznat kapitálový příjem v DP',
		simple:
			'Kapitálové příjmy je třeba přiznat v daňovém přiznání, pokud:\n\n- Zahraniční dividendy nebyly zdaněny českou srážkovou daní\n- Chcete započíst zahraniční daň\n- Příjem přesahuje limit pro osvobození\n- Zdrojem je P2P platforma či zahraniční broker bez české srážkové daně\n\nPříjmy již zdaněné českou srážkovou daní (např. CZ dividendy od českého brokera) přiznávat nemusíte.',
		legal:
			'Povinnost přiznat kapitálový příjem vyplývá z § 8 a § 38g zákona č. 586/1992 Sb. Příjmy zdaněné srážkovou daní dle § 36 se do základu daně nezahrnují (§ 36 odst. 7), pokud se poplatník nerozhodne je zahrnout (§ 36 odst. 7 věta druhá). Zahraniční příjmy se uvádějí vždy, zápočet daně dle § 38f a příslušné smlouvy o zamezení dvojího zdanění.'
	},
	'doplatek-preplatek': {
		title: 'Doplatek vs přeplatek',
		simple:
			'Výsledek daňového přiznání je buď doplatek, nebo přeplatek:\n\n- Doplatek: vaše daň je vyšší než zaplacené zálohy -- rozdíl musíte doplatit\n- Přeplatek: zaplatili jste na zálohách více, než činila vaše daň -- stát vám rozdíl vrátí\n\nDoplatek je splatný do lhůty pro podání daňového přiznání. O přeplatek musíte požádat (formulář "Žádost o vrácení přeplatku").',
		legal:
			'Splatnost daně z příjmů upravuje § 135 zákona č. 280/2009 Sb. (daňový řád) -- daň je splatná v poslední den lhůty pro podání přiznání. Přeplatek na dani vrací správce daně na základě žádosti do 30 dnů (§ 155 odst. 3 daňového řádu). Přeplatek menší než 200 Kč se nevrací (§ 155 odst. 2).'
	},
	'srazena-dan': {
		title: 'Srážená daň z kapitálu',
		simple:
			'Srážková daň je daň, kterou za vás strhne banka nebo broker ještě před výplatou. U českých dividend a úroků je to 15 %. Vy obdržíte částku již po zdanění.\n\nPříjem zdaněný srážkovou daní nemusíte uvádět v daňovém přiznání -- daň je již vypořádána. Výjimkou jsou zahraniční dividendy, kde může být srážková daň jiná a chcete ji započíst.',
		legal:
			'Srážková daň je upravena v § 36 zákona č. 586/1992 Sb. Sazba 15 % se uplatní u dividend, úroků z vkladů, úroků z dluhopisů a dalších příjmů z § 8 (§ 36 odst. 2). Plátcem srážkové daně je vyplácitel příjmu (§ 38d), který daň srazí a odvede do konce měsíce následujícího po měsíci sražení.'
	},
	'kurz-cnb': {
		title: 'Kurz ČNB pro přepočet',
		simple:
			'Zahraniční příjmy a výdaje se pro daňové účely přepočítávají na české koruny kurzem ČNB. Používá se kurz platný v den uskutečnění transakce (den obchodu, den výplaty dividendy).\n\nAplikace používá devizový kurz ČNB. U měn, které ČNB neuvádí přímo, se použije křížový kurz přes USD.',
		legal:
			'Přepočet cizího kurzu upravuje § 38 zákona č. 586/1992 Sb. Poplatník použije jednotný kurz stanovený GFŘ (roční průměrný kurz) nebo kurz devizového trhu ČNB platný v den uskutečnění transakce. Jednotný kurz vydává GFŘ v pokynu po skončení roku. Pro účely § 10 se běžně používá denní kurz ČNB.'
	},
	'nova-zaloha': {
		title: 'Nová měsíční záloha',
		simple:
			'Po podání přehledu ČSSZ a ZP vám pojišťovna vypočítá novou výši měsíční zálohy na další období. Výše zálohy se odvíjí od vašich příjmů v minulém roce.\n\nPokud jste měli vyšší příjmy, zálohy se zvýší. Pokud nižší, sníží se (ale ne pod zákonné minimum). Nová záloha platí od měsíce následujícího po podání přehledu.',
		legal:
			'Zálohy na sociální pojištění: § 14a zákona č. 589/1992 Sb. Nová výše zálohy = 1/12 ročního pojistného. Minimální záloha se odvíjí od průměrné mzdy. Platí od měsíce následujícího po měsíci podání přehledu. Zálohy na zdravotní pojištění: § 7 zákona č. 592/1992 Sb. Minimální záloha je 50 % z minimálního vyměřovacího základu.'
	},
	'fifo-prepocet': {
		title: 'FIFO metoda pro nabývací cenu',
		simple:
			'FIFO (First In, First Out) je metoda pro určení nabývací ceny při prodeji cenných papírů. Znamená, že při prodeji se jako první "spotřebují" nejstarší nakoupené kusy.\n\nPříklad: Koupili jste 10 ks za 100 Kč a pak 10 ks za 150 Kč. Pokud prodáte 10 ks, nabývací cena bude 100 Kč (použijí se první nakoupené kusy).\n\nFIFO metoda je pro OSVČ jediná povolená metoda.',
		legal:
			'FIFO metoda je jediná přípustná metoda oceňování pro fyzické osoby při prodeji cenných papírů dle § 10 odst. 4 zákona č. 586/1992 Sb. a pokynu GFŘ-D-22. Při FIFO se přiřadí výdaj k přímo identifikovatelnému nákupu, nebo se použije nejstarší nepřiřazený nákup. Náklady na poplatky brokera jsou součástí nabývací ceny.'
	},
};

// Dynamic topics with year-specific amounts.
// When TaxConstants are available, amounts are interpolated.
// Without constants, generic text is shown.
export function getHelpTopics(tc?: TaxConstants | null): Record<HelpTopicId, HelpTopic> {
	return {
		...(staticTopics as Record<HelpTopicId, HelpTopic>),

		'pausalni-vydaje': {
			title: 'Paušální výdaje',
			simple: tc
				? `Paušální výdaje jsou zjednodušený způsob uplatňování nákladů -- místo evidování každého výdaje si odečtete procento z příjmů. Procenta a maxima pro rok ${tc.year}:\n\n` +
					Object.entries(tc.flat_rate_caps)
						.sort(([a], [b]) => Number(b) - Number(a))
						.map(([pct, cap]) => `- ${pct} %: max ${fmtCZK(cap)}`)
						.join('\n') +
					'\n\nPaušální výdaje se hodí, pokud máte nízké skutečné náklady. Pozor: při paušálních výdajích nelze uplatnit slevu na manžela/ku ani daňové zvýhodnění na děti.'
				: 'Paušální výdaje jsou zjednodušený způsob uplatňování nákladů -- místo evidování každého výdaje si odečtete procento z příjmů. Každé procento má roční strop, který se může lišit podle roku.\n\nPaušální výdaje se hodí, pokud máte nízké skutečné náklady. Pozor: při paušálních výdajích nelze uplatnit slevu na manžela/ku ani daňové zvýhodnění na děti.',
			legal:
				'Paušální výdaje (výdaje procentem z příjmů) upravuje § 7 odst. 7 zákona č. 586/1992 Sb. Sazby: 80 % (zemědělství, řemesla), 60 % (živnost volná), 40 % (svobodná povolání), 30 % (nájem). Stropy se mění podle roku. Při paušálních výdajích nelze uplatnit slevu na manžela (§ 35ca) ani daňové zvýhodnění na děti (§ 35c odst. 9).'
		},

		'dan-15-23': {
			title: 'Sazba daně 15 % a 23 %',
			simple: tc
				? `Daň z příjmů fyzických osob má dvě sazby:\n\n- 15 % ze základu daně do ${fmtCZK(tc.progressive_threshold)}\n- 23 % z části základu daně nad ${fmtCZK(tc.progressive_threshold)}\n\nPráh ${fmtCZK(tc.progressive_threshold)} odpovídá 48násobku průměrné mzdy pro rok ${tc.year}. Většina OSVČ se vejde do 15% pásma.`
				: 'Daň z příjmů fyzických osob má dvě sazby:\n\n- 15 % ze základu daně do zákonem stanoveného prahu\n- 23 % z části základu daně nad tento práh\n\nPráh odpovídá 48násobku průměrné mzdy a mění se každý rok. Většina OSVČ se vejde do 15% pásma.',
			legal:
				'Sazby daně z příjmů fyzických osob upravuje § 16 zákona č. 586/1992 Sb. Základní sazba 15 % a solidární sazba 23 % z části základu daně přesahující 48násobek průměrné mzdy (§ 16 odst. 2). Průměrná mzda se stanoví dle § 21g.'
		},

		'sleva-na-poplatnika': {
			title: 'Základní sleva na dani',
			simple: tc
				? `Základní sleva na poplatníka je částka, kterou si každý automaticky odečte od vypočtené daně. Pro rok ${tc.year} činí ${fmtCZK(tc.basic_credit)} ročně (${fmtCZK(Math.round(tc.basic_credit / 12))} měsíčně).\n\nDíky této slevě neplatíte daň z prvních cca ${fmtCZK(Math.round(tc.basic_credit / 0.15))} zisku. Sleva se uplatní vždy v plné výši -- neproporcionalizuje se podle měsíců.`
				: 'Základní sleva na poplatníka je částka, kterou si každý automaticky odečte od vypočtené daně. Konkrétní výše závisí na zdaňovacím období.\n\nDíky této slevě neplatíte daň z určité části zisku. Sleva se uplatní vždy v plné výši -- neproporcionalizuje se podle měsíců.',
			legal:
				'Základní sleva na poplatníka je upravena v § 35ba odst. 1 písm. a) zákona č. 586/1992 Sb. Tuto slevu uplatňuje každý poplatník bez ohledu na výši příjmů. Na rozdíl od ostatních slev se neproporcionalizuje a uplatňuje se vždy v plné roční výši.'
		},

		'zvyhodneni-na-deti': {
			title: 'Daňové zvýhodnění na děti',
			simple: tc
				? `Daňové zvýhodnění na děti je částka, kterou si odečtete od daně za každé vyživované dítě. Roční částky (${tc.year}):\n\n- 1. dítě: ${fmtCZK(tc.child_benefit_1)}\n- 2. dítě: ${fmtCZK(tc.child_benefit_2)}\n- 3. a další: ${fmtCZK(tc.child_benefit_3_plus)}\n\nPokud je dítě držitelem ZTP/P, částka se zdvojnásobuje. Zvýhodnění může vytvořit "daňový bonus" -- pokud je vyšší než vaše daň, stát vám rozdíl vrátí (max ${fmtCZK(tc.max_child_bonus)}/rok).`
				: 'Daňové zvýhodnění na děti je částka, kterou si odečtete od daně za každé vyživované dítě. Konkrétní výše se liší podle pořadí dítěte a zdaňovacího období.\n\nPokud je dítě držitelem ZTP/P, částka se zdvojnásobuje. Zvýhodnění může vytvořit "daňový bonus" -- pokud je vyšší než vaše daň, stát vám rozdíl vrátí.',
			legal:
				'Daňové zvýhodnění na vyživované dítě upravuje § 35c zákona č. 586/1992 Sb. Částky se mění podle roku. U dítěte s ZTP/P se částky zdvojnásobují (§ 35c odst. 1). Maximální roční daňový bonus je stanoven v § 35c odst. 3. Zvýhodnění nelze uplatnit při paušálních výdajích (§ 35c odst. 9).'
		},

		'nezdanitelne-odpocty': {
			title: 'Odpočty ze základu daně',
			simple: tc
				? `Nezdanitelné části základu daně jsou částky, které si odečtete od základu daně PŘED výpočtem daně (na rozdíl od slev, které se odečítají od daně samotné). Patří sem:\n\n- Úroky z hypotéky (max ${fmtCZK(tc.deduction_cap_mortgage)}/rok)\n- Penzijní spoření (max ${fmtCZK(tc.deduction_cap_pension)}/rok)\n- Životní pojištění (max ${fmtCZK(tc.deduction_cap_life_insurance)}/rok)\n- Dary (max 15 % základu daně)\n- Odborové příspěvky (max ${fmtCZK(tc.deduction_cap_union)}/rok)`
				: 'Nezdanitelné části základu daně jsou částky, které si odečtete od základu daně PŘED výpočtem daně (na rozdíl od slev, které se odečítají od daně samotné). Patří sem úroky z hypotéky, penzijní spoření, životní pojištění, dary a odborové příspěvky. Konkrétní stropy závisí na zdaňovacím období.',
			legal:
				'Nezdanitelné části základu daně upravuje § 15 zákona č. 586/1992 Sb. Úroky z úvěru na bydlení (§ 15 odst. 3). Penzijní připojištění/spoření (§ 15 odst. 5): částka nad 12 000 Kč. Soukromé životní pojištění (§ 15 odst. 6). Dary na veřejně prospěšné účely (§ 15 odst. 1): min 2 % základu daně nebo 1 000 Kč, max 15 %. Stropy se mění podle roku.'
		},

		ztpp: {
			title: 'ZTP/P -- zvláště těžké postižení s průvodcem',
			simple: tc
				? `ZTP/P je průkaz pro osoby se zvláště těžkým zdravotním postižením. V kontextu daní má ZTP/P vliv na:\n\n- Sleva na manžela/ku se zdvojnásobuje (z ${fmtCZK(tc.spouse_credit)} na ${fmtCZK(tc.spouse_credit * 2)})\n- Daňové zvýhodnění na dítě se zdvojnásobuje\n\nZTP/P status se prokazuje průkazem vydaným Úřadem práce ČR.`
				: 'ZTP/P je průkaz pro osoby se zvláště těžkým zdravotním postižením. V kontextu daní má ZTP/P vliv na:\n\n- Sleva na manžela/ku se zdvojnásobuje\n- Daňové zvýhodnění na dítě se zdvojnásobuje\n\nZTP/P status se prokazuje průkazem vydaným Úřadem práce ČR.',
			legal:
				'Držitel průkazu ZTP/P je definován v § 34 zákona č. 329/2011 Sb. o poskytování dávek osobám se zdravotním postižením. Zdvojnásobení slevy na manžela/ku: § 35ba odst. 1 písm. b) zákona č. 586/1992 Sb. Zdvojnásobení zvýhodnění na dítě: § 35c odst. 1 téhož zákona.'
		},

		'sleva-na-manzela': {
			title: 'Sleva na manžela/ku',
			simple: tc
				? `Slevu na manžela/ku si můžete uplatnit, pokud váš manžel/ka měl/a za zdaňovací období vlastní roční příjmy nepřesahující ${fmtCZK(tc.spouse_income_limit)}. Do těchto příjmů se nezapočítávají např. rodičovský příspěvek, porodné, dávky státní sociální podpory či stipendia.\n\nSleva činí ${fmtCZK(tc.spouse_credit)} ročně. Pokud je manžel/ka držitelem ZTP/P, sleva se zdvojnásobuje na ${fmtCZK(tc.spouse_credit * 2)}. Sleva se proporcionalizuje podle měsíců -- počítá se od měsíce, na jehož počátku byly podmínky splněny.\n\nDůležité: slevu na manžela/ku NELZE uplatnit, pokud používáte paušální výdaje.`
				: 'Slevu na manžela/ku si můžete uplatnit, pokud váš manžel/ka měl/a za zdaňovací období nízké vlastní roční příjmy. Do těchto příjmů se nezapočítávají např. rodičovský příspěvek, porodné či stipendia.\n\nKonkrétní výše slevy a limit příjmů závisí na zdaňovacím období. Pokud je manžel/ka držitelem ZTP/P, sleva se zdvojnásobuje. Sleva se proporcionalizuje podle měsíců.\n\nDůležité: slevu na manžela/ku NELZE uplatnit, pokud používáte paušální výdaje.',
			legal:
				'Sleva na manžela/ku je upravena v § 35ba odst. 1 písm. b) zákona č. 586/1992 Sb. Podmínka: manžel/ka žijící ve společné domácnosti s vlastním ročním příjmem nepřesahujícím zákonem stanovený limit. Do vlastního příjmu se nezapočítávají dávky dle § 35ba odst. 1 písm. b).\n\nU držitele ZTP/P se sleva zdvojnásobuje. Proporcionalizace dle § 35ba odst. 3 -- 1/12 za každý měsíc, na jehož počátku byly podmínky splněny. Při paušálních výdajích (§ 7 odst. 7) nelze slevu uplatnit (§ 35ca).'
		}
	};
}

// Backward-compatible export for non-tax pages (no year context).
export const helpTopics: Record<HelpTopicId, HelpTopic> = getHelpTopics();
