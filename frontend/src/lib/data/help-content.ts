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
	| 'rocni-dane';

export interface HelpTopic {
	title: string;
	simple: string;
	legal: string;
}

export const helpTopics: Record<HelpTopicId, HelpTopic> = {
	'variabilni-symbol': {
		title: 'Variabilni symbol',
		simple:
			'Variabilni symbol je cislo, ktere identifikuje platbu. Kdyz vam nekdo posle penize na ucet, banka podle variabilniho symbolu pozna, ke ktere fakture platba patri.\n\nVetsinou se pouziva cislo faktury nebo jeho cast. Dulezite je, aby kazda faktura mela unikatni variabilni symbol -- jinak nepoznate, kdo za co platil.',
		legal:
			'Variabilni symbol je numericke pole o maximalni delce 10 cislic. Je definovan vyhlaskou CNB c. 169/2011 Sb. jako identifikator transakce v tuzemskem platebnim styku.\n\nPodle zakona c. 284/2009 Sb. o platebnim styku je variabilni symbol soucast platebniho prikazu a slouzi k identifikaci platby mezi platcem a prijemcem. Neni povinny ze zakona, ale je standardni soucasti fakturacni praxe v CR.'
	},
	'konstantni-symbol': {
		title: 'Konstantni symbol',
		simple:
			'Konstantni symbol je cislo, ktere rika, o jaky typ platby se jedna (napr. platba za zbozi, sluzby, najem). V praxi se dnes pouziva minimalne -- vetsina bank ho nevyzaduje a pro OSVC neni potrebny.\n\nPokud si nejste jisti, muzete pole nechat prazdne.',
		legal:
			'Konstantni symbol je definovan vyhlaskou CNB c. 169/2011 Sb. Jedna se o ctyrciselny kod charakterizujici platbu z hlediska jejiho ucelu. Od roku 2004 neni jeho uvadeni povinne pro bezne platby.\n\nNejcastejsi hodnoty: 0008 (platba za zbozi), 0308 (platba za sluzby), 0558 (ostatni bezhotovostni platby).'
	},
	duzp: {
		title: 'Datum uskutecneni zdanitelneho plneni (DUZP)',
		simple:
			'DUZP je datum, kdy skutecne doslo k dodani zbozi nebo poskyteni sluzby. Ne kdy jste vystavili fakturu, ne kdy vam prisly penize -- ale kdy jste realne odvedli praci nebo dodali produkt.\n\nNapr. pokud jste programovali web cely leden a fakturujete az 5. unora, DUZP bude posledni den, kdy jste praci predali (treba 31. ledna).\n\nPro platce DPH je DUZP klicove, protoze urcuje, do ktereho zdanovaciho obdobi faktura patri.',
		legal:
			'DUZP je definovano v zakone c. 235/2004 Sb. o DPH, § 21. U dodani zbozi je to den dodani (§ 21 odst. 1). U poskytovani sluzeb den poskytnuti nebo den vystaveni danoveho dokladu, pokud nastal drive (§ 21 odst. 3).\n\nPlatce DPH je povinen priznat dan na vystupu ke dni uskutecneni zdanitelneho plneni (§ 20a). DUZP urcuje zdanovaci obdobi, ve kterem musi byt dan odvedena.'
	},
	'datum-splatnosti': {
		title: 'Datum splatnosti',
		simple:
			'Datum splatnosti je den, do ktereho ma odberatel zaplatit fakturu. Pokud zakaznik nezaplati do tohoto data, faktura je "po splatnosti" a muzete uplatnovat uroky z prodleni.\n\nBezna splatnost je 14 nebo 30 dni od data vystaveni. Muze byt i delsi -- zalezi na dohode s odberatelem.',
		legal:
			'Splatnost je smluvni ujednani dle zakona c. 89/2012 Sb. (obcansky zakonik), § 1958-1964. Pokud neni dohodnuta, je splatnost bez zbytecneho odkladu po doruceni faktury.\n\nPodle zakona c. 340/2015 Sb. o registru smluv a § 1963 obcanskeho zakoniku plati pro vztahy s verejnym sektorem maximalni splatnost 30 dni. Pro obchodni vztahy mezi podnikateli je smluvni splatnost maximalne 60 dni (§ 1963a OZ), pokud to neni vuci veriteli hrube nespravedlive.'
	},
	'zpusob-platby': {
		title: 'Zpusob platby',
		simple:
			'Zpusob platby urcuje, jak odberatel zaplati fakturu. Nejcasteji bankovnim prevodem -- v tom pripade faktura obsahuje cislo uctu a variabilni symbol.\n\nDalsi moznosti jsou hotovost, platba kartou nebo dobirka. Pro ucetni a danove ucely je dulezite, aby zpusob platby odpovidal realite.',
		legal:
			'Zpusob platby na fakture neni povinnou nalezitosti danoveho dokladu dle § 29 zakona c. 235/2004 Sb. o DPH. Jedna se vsak o beznou obchodni nalezitost.\n\nPro hotovostni platby plati limit 270 000 Kc dle zakona c. 254/2004 Sb. o omezeni plateb v hotovosti (§ 4). Poruseni je spravni delikt s pokutou do 500 000 Kc pro fyzicke osoby.'
	},
	'poznamka-faktura': {
		title: 'Poznamka na fakture',
		simple:
			'Text, ktery se zobrazi primo na fakture, kterou poslete zakaznikovi. Muzete sem napsat napr. podekovan za spolupraci, informaci o probihajici akci nebo upozorneni na zmenu bankovniho uctu.\n\nTato poznamka je viditelna pro odberatele.',
		legal:
			'Poznamka na fakture neni povinnou nalezitosti danoveho dokladu dle § 29 zakona c. 235/2004 Sb. Pokud vsak slouzi jako informace o osvobozenem plneni, musi obsahovat odkaz na prislusne ustanoveni zakona (§ 29 odst. 2 pism. c).\n\nNapr. u osvobozenych plneni: "Oslobozeno od DPH dle § 51 zakona c. 235/2004 Sb."'
	},
	'poznamka-interni': {
		title: 'Interni poznamka',
		simple:
			'Soukroma poznamka, kterou vidite jen vy. Na fakture se nezobrazuje. Muzete sem napsat cokoli pro vlastni evidenci -- napr. "dohodnuto s Petrem 15.3.", "sleva za doporuceni" apod.',
		legal: 'Interni poznamka nema pravni relevanci a neobjevuje se na zadnem dokladu. Slouzi pouze pro interni evidenci podnikatele.'
	},
	'qr-platba': {
		title: 'QR platba',
		simple:
			'QR kod na fakture umozni odberateli naskenovat platbu mobilem. Po naskenovani se v bankovni aplikaci automaticky predvyplni cislo uctu, castka, variabilni symbol a dalsi udaje.\n\nOdberatel tak nemusi nic opisovat a platba probehne bez chyb. QR platba je standard Ceske bankovni asociace.',
		legal:
			'QR platba (SPD -- Short Payment Descriptor) je standard Ceske bankovni asociace pro mobilni platby. Format je definovan specifikaci CBA a je podporovan vsemi hlavnimi bankami v CR.\n\nFormat QR kodu: SPD*1.0*ACC:{IBAN}*AM:{castka}*CC:CZK*X-VS:{variabilni symbol}*...'
	},
	'danove-uznatelny': {
		title: 'Danove uznatelny naklad',
		simple:
			'Danove uznatelny naklad je vydaj, ktery si muzete odecist od prijmu a tim snizit dan z prijmu. Musi splnovat podminku: byl vynalozen na dosazeni, zajisteni a udrzeni vasich prijmu.\n\nPriklad: Notebook pro praci = danove uznatelny. Dovolena = neni danove uznatelna.\n\nPokud pouzivate skutecne vydaje (ne pausalni), je dulezite spravne oznacit, ktere naklady jsou danove uznatelne.',
		legal:
			'Danove uznatelne naklady jsou definovany v § 24 zakona c. 586/1992 Sb. o danich z prijmu. Jsou to vydaje vynalozene na dosazeni, zajisteni a udrzeni zdanitelnych prijmu.\n\nDanove neuznatelne naklady vycte § 25 tehoz zakona (napr. repre, pokuty, penale). Preukazuji se daovymi doklady -- podnikatel musi byt schopen prokazat ucetni doklad, ucel vydaje a souvislost s podnikatelskou cinnosti.'
	},
	'podil-podnikani': {
		title: 'Podil pro podnikani',
		simple:
			'Nektery naklady pouzivate jak pro podnikani, tak pro osobni ucely. Napr. auto, telefon nebo internet. Podil pro podnikani urcuje, kolik procent nakladu uplatnite jako danovy vydaj.\n\nPriklad: Telefon pouzivate z 60 % pro praci a z 40 % soukrome. Podil pro podnikani je 60 % a jako danovy naklad si uplatnite 60 % z ceny.\n\nPomerne je treba rozdelit i DPH na vstupu, pokud jste platce DPH.',
		legal:
			'Kraceni nakladu u majetku pouzivaneho i pro soukrome ucely upravuje § 24 odst. 2 pism. h) zakona c. 586/1992 Sb. Naklady se uplatnuji v pomerne vysi odpovidajici rozsahu pouziti pro podnikatelskou cinnost.\n\nPro DPH plati narok na odpocet v pomerne vysi dle § 75 zakona c. 235/2004 Sb. Podnikatel je povinen vest evidenci pouziti majetku pro podnikatelske a soukrome ucely.'
	},
	'sazba-dph': {
		title: 'Sazba DPH',
		simple:
			'Sazba DPH (dan z pridane hodnoty) urcuje, kolik procent dane se prida k cene zbozi ci sluzby. V Cesku jsou aktualne dve sazby:\n\n- 21 % -- zakladni sazba (vetsina zbozi a sluzeb)\n- 12 % -- snizena sazba (potraviny, leky, knihy, ubytovani, stavebni prace)\n\nPokud nejste platce DPH, DPH neuctujete a na fakture uvedete 0 %.',
		legal:
			'Sazby DPH stanovuje § 47 zakona c. 235/2004 Sb. o DPH. Od 1. 1. 2024 plati dve sazby: zakladni 21 % a snizena 12 % (slouceni puvodnich dvou snizenych sazeb 15 % a 10 %).\n\nZarazeni zbozi a sluzeb do snizene sazby je v priloze c. 2, 3 a 3a tehoz zakona. Neplatce DPH neni opravnen vyuctovat dan a nesmi ji uvest na dokladu (§ 26 odst. 3).'
	},
	'cislo-dokladu': {
		title: 'Cislo dokladu',
		simple:
			'Cislo dokladu jednoznacne identifikuje ucetni doklad (fakturu, uctenku, pokladni doklad). Slouzi pro evidenci -- abyste kazdy naklad snadno dohledali.\n\nMuze to byt cislo z prijate faktury od dodavatele, nebo vase vlastni cislo, pokud doklad nemitate (napr. pokladni blok oznacite "P-001").',
		legal:
			'Poradove cislo dokladu je povinnou nalezitosti danoveho dokladu dle § 29 odst. 1 pism. b) zakona c. 235/2004 Sb. Musi byt prirazeno v ramci jedne ci vice ciselnych rad, ktere zarucuji jeho jednoznacnost.\n\nI pro neplatce DPH je jednoznacna identifikace dokladu povinnosti dle § 11 zakona c. 563/1991 Sb. o ucetnictvi.'
	},
	ico: {
		title: 'Identifikacni cislo osoby (ICO)',
		simple:
			'ICO je osmimistne cislo, ktere dostane kazdy podnikatel nebo firma pri registraci. Slouzi k jednoznacne identifikaci -- jako "rodne cislo" pro podnikani.\n\nICO se uvadi na vsech fakturach a obchodnich dokumentech. Podle ICO si muzete overit odberatele v obchodnim rejstriku nebo registru ARES.',
		legal:
			'ICO je definovano zakonem c. 111/2009 Sb. o zakladnich registrech, § 24-26. Prideluje ho registracni organ (zivnostensky urad, rejstrikovy soud).\n\nPodle § 29 odst. 1 pism. a) zakona c. 235/2004 Sb. je ICO povinnou nalezitosti danoveho dokladu. Povinnost uvadet ICO na obchodnich listinach plyne take z § 435 zakona c. 89/2012 Sb. (obcansky zakonik).'
	},
	dic: {
		title: 'Danove identifikacni cislo (DIC)',
		simple:
			'DIC je cislo, ktere identifikuje platce dane. V Cesku ma format "CZ" + ICO (napr. CZ12345678). DIC dostanete po registraci k DPH u financniho uradu.\n\nPokud nejste platce DPH, DIC nemusite uvadet. Pokud jste platce, je DIC povinne na kazde fakture.',
		legal:
			'DIC je definovano v § 130 zakona c. 280/2009 Sb. (danovy rad). Pro ucely DPH je upraveno v § 4a zakona c. 235/2004 Sb. -- u fyzickych osob ma format "CZ" + rodne cislo, u pravnickych osob "CZ" + ICO.\n\nDIC je povinnou nalezitosti danoveho dokladu dle § 29 odst. 1 pism. a) zakona c. 235/2004 Sb. pro platce DPH.'
	},
	ares: {
		title: 'ARES -- Administrativni registr ekonomickych subjektu',
		simple:
			'ARES je verejny registr, kde si muzete overit udaje o jakemkoli podnikateli nebo firme v Cesku. Staci zadat ICO a zjistite nazev, sidlo, pravni formu a dalsi informace.\n\nV ZFaktury se ARES pouziva pro automaticke doplneni udaju o odberateli -- zadejte ICO a system stahne jmeno a adresu automaticky.',
		legal:
			'ARES je informacni system verejne spravy provozovany Ministerstvem financi CR. Agreguje data z vice registru: obchodniho rejstriku, zivnostenskeho rejstriku, registru DPH a dalsich.\n\nPristup k datum je bezuplatny a verejny dle zakona c. 106/1999 Sb. o svobodnem pristupu k informacim. API ARES je dostupne na ares.gov.cz.'
	},
	iban: {
		title: 'IBAN -- Mezinarodni cislo bankovniho uctu',
		simple:
			'IBAN je mezinarodni format cisla bankovniho uctu. V Cesku zacina "CZ" a ma celkem 24 znaku (napr. CZ65 0800 0000 1920 0014 5399).\n\nIBAN se pouziva pro zahranicni platby, ale stale casteji i pro tuzemske. Na fakture ho uvadejte, pokud mate zahranicni odberatele nebo pokud chcete uzivateli usnadnit platbu QR kodem.',
		legal:
			'IBAN je standardizovan normou ISO 13616. V CR je povinny pro prehranicni platby v ramci EU/EHP dle narizeni EP a Rady (EU) c. 260/2012 (SEPA narizeni).\n\nPro tuzemske platby IBAN povinny neni, ale banky ho podporuji a je soucasti QR platebniho formatu CBA. Cesky IBAN ma format: CZ + 2 kontrolni cislice + 4 cislice kod banky + 16 cislic cislo uctu.'
	},
	'swift-bic': {
		title: 'SWIFT/BIC kod',
		simple:
			'SWIFT kod (take BIC) identifikuje banku pri mezinarodnich platbach. Je to 8 nebo 11 znaku dlouhy kod (napr. KOMBCZPP pro Komercni banku).\n\nUvadejte ho na fakturach pro zahranicni odberatele -- bez SWIFT kodu nemuze platba ze zahranici dojit na spravnou banku.',
		legal:
			'SWIFT (Society for Worldwide Interbank Financial Telecommunication) kod, formalne BIC (Bank Identifier Code), je standardizovan normou ISO 9362.\n\nPro platby v ramci SEPA (Single Euro Payments Area) neni BIC povinny od 1. 2. 2016 dle narizeni (EU) c. 260/2012. Pro platby mimo SEPA je BIC stale nutny pro spravne smerovani platby.'
	},
	'platce-dph': {
		title: 'Platce DPH',
		simple:
			'Platce DPH je podnikatel registrovany k dani z pridane hodnoty. Musi k cenam svych sluzeb a zbozi pricitovat DPH a odvadet ho statu. Na druhou stranu si muze odpocist DPH z nakupu souvisejicich s podnikanim.\n\nPovinne se platcem DPH stavate, kdyz vas obrat za 12 po sobe jdoucich mesicu prekroci 2 miliony Kc. Muze se stat i dobrovolne.',
		legal:
			'Registrace platce DPH je upravena v § 6-6f zakona c. 235/2004 Sb. Povinnou registraci vyvola prekroceni obratu 2 000 000 Kc za 12 po sobe jdoucich kalendarnich mesicu (§ 6 odst. 1) -- platnost od 1. 1. 2025.\n\nPlatce je povinen podavat danove priznani (§ 101), kontrolni hlaseni (§ 101c) a v nekterych pripadech souhrnne hlaseni (§ 102). Dan se odvadi mesicne nebo ctvrtletne dle § 99-99a.'
	},
	'priznani-dph': {
		title: 'Priznani k DPH',
		simple:
			'Priznani k DPH je formular, ktery platce DPH odevzdava financnimu uradu. Obsahuje prehled vasi dane na vystupu (DPH z vasich faktur) a dane na vstupu (DPH z vasich nakupu). Rozdil bud zaplatite statu, nebo vam stat vrati.\n\nPodava se mesicne nebo ctvrtletne, vzdy do 25. dne nasledujiciho mesice.',
		legal:
			'Danove priznani k DPH upravuje § 101 zakona c. 235/2004 Sb. Platce je povinen podat priznani do 25 dni po skonceni zdanovaciho obdobi (§ 101 odst. 1).\n\nZdanovaci obdobi je kalendarni mesic nebo ctvrtleti (§ 99-99a). Priznani se podava elektronicky ve formatu XML na portal financni spravy (EPO). Vzor formulare stanovi Ministerstvo financi vyhlaskou.'
	},
	'kontrolni-hlaseni': {
		title: 'Kontrolni hlaseni',
		simple:
			'Kontrolni hlaseni je mesicni report pro financni urad, ktery obsahuje rozpis vsech vasich faktur (vydanych i prijatych) s DPH. Slouzi statu ke krizove kontrole -- oveuje, ze DPH, ktere vy uctujete na vystupu, si vas odberatel uplatnil na vstupu, a naopak.\n\nPodava se vzdy do 25. dne nasledujiciho mesice. Fyzicke osoby mohou podavat ctvrtletne.',
		legal:
			'Kontrolni hlaseni je upraveno v § 101c-101i zakona c. 235/2004 Sb. Podava se elektronicky ve formatu XML.\n\nLhuty: pravnicke osoby mesicne, fyzicke osoby ve lhute pro podani danoveho priznani (§ 101e). Za nepodani hrozi pokuta 10 000-50 000 Kc (§ 101h). Za nepodani na vyzvu az 500 000 Kc.\n\nObsahuje udaje o prijatych a uskutecnenych plnenich nad 10 000 Kc vcetne DPH s identifikaci obchodniho partnera (DIC).'
	},
	'souhrnne-hlaseni': {
		title: 'Souhrnne hlaseni',
		simple:
			'Souhrnne hlaseni podavate, pokud dodavate zbozi nebo sluzby do jinych zemi EU platcum DPH. Hlaseni informuje financni urad o techto dodavkach.\n\nPokud obchodujete pouze v Cesku, souhrnne hlaseni vas nezajima.',
		legal:
			'Souhrnne hlaseni upravuje § 102 zakona c. 235/2004 Sb. Podava se za kazdy kalendarni mesic (pri dodani zbozi) nebo ctvrtleti (pri poskytovani sluzeb) do 25 dni po skonceni obdobi.\n\nTyka se dodani zbozi do jineho clenskeho statu osobe registrovane k DPH (§ 102 odst. 1 pism. a), poskytovani sluzeb s mistem plneni v jinem clenskem state (§ 102 odst. 1 pism. d) a premisteni obchodniho majetku (§ 102 odst. 1 pism. b).'
	},
	'typ-podani': {
		title: 'Typ podani',
		simple:
			'Typ podani urcuje, zda se jedna o radne, opravne nebo dodatecne podani:\n\n- Radne -- prvni podani za dane obdobi\n- Opravne -- oprava podani pred uplynutim lhuty (nahradi puvodni)\n- Dodatecne -- oprava po uplynnuti lhuty (podava se navic k radnemu)',
		legal:
			'Typy podani definuje zakon c. 280/2009 Sb. (danovy rad):\n\n- Radne podani (§ 135) -- standardni podani v zakonnem terminu\n- Opravne podani (§ 138) -- nahrazuje puvodni podani pred uplynutim lhuty, posledni podane plati\n- Dodatecne podani (§ 141) -- podava se po uplynnuti lhuty pro radne podani, pokud podnikatel zjisti chybu. Lhuta pro podani: do konce mesice nasledujiciho po zjisteni chyby'
	},
	'ciselne-rady': {
		title: 'Ciselne rady',
		simple:
			'Ciselne rady zajistuji automaticke cislovani vasich faktur. Misto rucniho zadavani cisel system sam priradi dalsi cislo v poradi.\n\nMuzete mit vice ciselnych rad -- napr. jednu pro tuzemske faktury (FV-2024-001) a jinou pro zahranicni (ZF-2024-001).',
		legal:
			'Povinnost ciselnych rad vyplyva z § 29 odst. 1 pism. b) zakona c. 235/2004 Sb. -- danovy doklad musi obsahovat poradove cislo prirazene v ramci jedne ci vice ciselnych rad.\n\nCiselna rada musi zarucovat jednoznacnost dokladu. Podnikatel je povinen vest evidenci vydanych dokladu a jejich ciselnych rad pro ucely pripadne kontroly financnim uradem.'
	},
	'prefix-format': {
		title: 'Prefix a format ciselne rady',
		simple:
			'Prefix je text pred cislem faktury (napr. "FV" pro fakturu vydanou). Format urcuje, jak bude cislo vypadat -- napr. "{prefix}{year}-{number:4}" vytvori cisla jako FV2024-0001, FV2024-0002 atd.\n\nCislovani se resetuje na zacatku kazdeho roku, takze prvni faktura noveho roku bude vzdy 0001.',
		legal:
			'Format ciselne rady neni zakonem predepsan. Zakon c. 235/2004 Sb. v § 29 vyzaduje pouze to, aby poradove cislo bylo jednoznacne v ramci ciselne rady.\n\nDoporucuje se vcetne roku (napr. 2024-001) pro snazsi orientaci a prukaznost pri danove kontrole. Prefix pomaha rozlisit typ dokladu (faktury vydane, prijate, dob ropisy atd.).'
	},
	'prijmy-naklady': {
		title: 'Prijmy a naklady',
		simple:
			'Prijmy jsou penize, ktere vam zakaznici zaplatili za vase sluzby nebo zbozi. Naklady jsou penize, ktere jste utratili v souvislosti s podnikanim.\n\nRozdil mezi prijmy a naklady je zakladem dane -- cim vice nakladu (danove uznatelnych) mate, tim mene dane zaplatite.',
		legal:
			'Prijmy z podnikani OSVC jsou upraveny v § 7 zakona c. 586/1992 Sb. o danich z prijmu. Zaklad dane se stanovi jako rozdil mezi prijmy a vydaji (§ 23).\n\nAlternativne muze OSVC uplatnit pausalni vydaje (§ 7 odst. 7): 80 % u remeslnych zivnosti, 60 % u ostatnich zivnosti, 40 % u prijmu z jineho podnikani. Pausalni vydaje jsou omezeny castkou 1 600 000 / 1 200 000 / 800 000 Kc.'
	},
	'neuhrazene-faktury': {
		title: 'Neuhrazene faktury',
		simple:
			'Neuhrazene faktury jsou faktury, ktere jste vystavili, ale zakaznik je jeste nezaplatil. Mohou byt pred splatnosti (zakaznik ma jeste cas) nebo po splatnosti (zakaznik je v prodleni).\n\nJe dulezite sledovat neuhrazene faktury a vcas upominat dluzniky. Po splatnosti maji uroky z prodleni.',
		legal:
			'Neuhrazene pohledavky po splatnosti lze danove odepisat dle § 24 odst. 2 pism. y) zakona c. 586/1992 Sb. u pohledavek za dluznikem v insolvencnim rizeni.\n\nOpravne polozky k pohledavkam upravuje zakon c. 593/1992 Sb. o rezervach: po 18 mesicich po splatnosti az 50 %, po 30 mesicich az 100 % (§ 8a). Uroky z prodleni se ridi § 1970 obcanskeho zakoniku -- repo sazba CNB + 8 p.b.'
	},
	'faktury-po-splatnosti': {
		title: 'Faktury po splatnosti',
		simple:
			'Faktura je po splatnosti, kdyz zakaznik nezaplatil do data splatnosti. Od tohoto okamziku je v prodleni a vy muvete uplatnit uroky z prodleni.\n\nDoporuceny postup: po 7 dnech prvni upominka, po 14 dnech druha upominka, po 30 dnech predsoudni upominka s vyhruzkou pravnimi kroky.',
		legal:
			'Prodleni dluznika upravuje § 1968-1975 zakona c. 89/2012 Sb. (obcansky zakonik). Dluznik, ktery svuj dluh radne a vcas neplni, je v prodleni (§ 1968).\n\nVeritel ma pravo na uroky z prodleni (§ 1970) ve vysi repo sazby CNB + 8 procentnich bodu. U obchodnich vztahu ma veritel take pravo na minimalni pausal 1 200 Kc za naklady spojene s uplatnenim pohledavky (narizeni vlady c. 351/2013 Sb.).'
	},
	'frekvence-opakovani': {
		title: 'Frekvence opakovani',
		simple:
			'Frekvence urcuje, jak casto se opakujici faktura automaticky vytvori. Napr. mesicni frekvence znamena, ze se faktura vytvori jednou mesicne.\n\nBezne frekvence: mesicni (napr. pausalni sluzby, najem), ctvrtletni (napr. pravidelne konzultace), rocni (napr. licence, predplatne).',
		legal:
			'Opakujici se plneni (trvalane plneni) je upraveno v § 21 odst. 8 zakona c. 235/2004 Sb. U opakovaneho plneni se DUZP stanovi nejpozdeji poslednim dnem zdanovaciho obdobi.\n\nSmlouvy na opakovane plneni (napr. najem, servisni smlouvy) se ridi ustanovenimi o zavazkovem pravu v obcanskem zakoniku (§ 1724 a nasl. zakona c. 89/2012 Sb.).'
	},
	'vystupni-dph': {
		title: 'Vystupni DPH',
		simple:
			'Vystupni DPH je dan, kterou uctujete svym zakaznikum na fakturach. Kdyz vystavite fakturu s DPH, tuto dan musíte odvest statu.\n\nNapr. fakturujete sluzbu za 10 000 Kc + 21 % DPH = 12 100 Kc. Tech 2 100 Kc je vystupni DPH, ktere odvedete financnimu uradu.',
		legal:
			'Vystupni DPH (dan na vystupu) je definovano v § 4 odst. 1 pism. c) zakona c. 235/2004 Sb. o DPH. Platce je povinen priznat dan na vystupu ke dni uskutecneni zdanitelneho plneni (§ 20a) nebo ke dni prijeti uhrady, pokud nastala drive (§ 21).\n\nDan na vystupu se uvadi v daovem priznani v radcich 1-13 formulare.'
	},
	'vstupni-dph': {
		title: 'Vstupni DPH',
		simple:
			'Vstupni DPH je dan, kterou jste zaplatili pri svych nakupech. Tuto dan si muzete odecist od vystupniho DPH -- tim snizite castku, kterou odvedete statu.\n\nNapr. koupite notebook za 24 200 Kc (20 000 + 4 200 DPH). Tech 4 200 Kc je vstupni DPH, ktere si odectete.',
		legal:
			'Narok na odpocet dane na vstupu upravuji § 72-73 zakona c. 235/2004 Sb. Platce ma narok na odpocet dane u prijatych zdanitelnych plneni, ktera pouzije pro uskutecneni sve ekonomicke cinnosti (§ 72 odst. 1).\n\nPodminkou odpoctu je drzeni danoveho dokladu (§ 73 odst. 1). Narok na odpocet lze uplatnit nejdrive za zdanovaci obdobi, ve kterem jsou splneny podminky (§ 73 odst. 3).'
	},
	'preneseni-danove-povinnosti': {
		title: 'Preneseni danove povinnosti',
		simple:
			'Preneseni danove povinnosti (reverse charge) znamena, ze DPH neplati dodavatel, ale odberatel. Dodavatel vystavi fakturu bez DPH a odberatel si DPH sam vypocita a prizna.\n\nPouziva se napr. u stavebnich praci, dodani srot a odpadu, nebo u obchodu mezi firmami v ramci EU.',
		legal:
			'Preneseni danove povinnosti (rezim reverse charge) upravuje § 92a zakona c. 235/2004 Sb. U tuzemskych plneni se tyka zbozi a sluzeb uvedenych v priloze c. 6 zakona (stavebni prace, srot, odpady aj.).\n\nPri preneseni danove povinnosti je odberatel povinen dan priznat a ma narok na odpocet (§ 92a odst. 1). Dodavatel uvede plneni v radku 25 danoveho priznani.'
	},
	'nadmerny-odpocet': {
		title: 'Nadmerny odpocet / Danova povinnost',
		simple:
			'Vysledek DPH priznani je bud danova povinnost, nebo nadmerny odpocet:\n\n- Danova povinnost: vystupni DPH > vstupni DPH -- rozdil zaplatite statu\n- Nadmerny odpocet: vstupni DPH > vystupni DPH -- stat vam vrati rozdil\n\nNadmerny odpocet vznika napr. pri velkych investicich (nakup stroje, rekonstrukce).',
		legal:
			'Nadmerny odpocet je definovan v § 4 odst. 1 pism. d) zakona c. 235/2004 Sb. Vznikne-li nadmerny odpocet, vrati ho spravce dane platci do 30 dni od vymereni (§ 105 odst. 1).\n\nSpravce dane muze pred vracenim zahajit postup k odstraneni pochybnosti (§ 89 danoveho radu), cimz se lhuta prodlouzi. Nadmerny odpocet se prednostne pouzije na uhradu pripadnych danových nedoplatku (§ 105 odst. 2).'
	},
	'zaklad-dane': {
		title: 'Zaklad dane',
		simple:
			'Zaklad dane je castka bez DPH, ze ktere se DPH vypocita. Napr. pokud je cena sluzby 12 100 Kc vcetne 21 % DPH, zaklad dane je 10 000 Kc a DPH 2 100 Kc.\n\nV DPH priznani se zaklad dane uvadi ve sloupcich vedle vypoctene dane.',
		legal:
			'Zaklad dane je definovan v § 36 zakona c. 235/2004 Sb. Zakladem dane je vse, co jako uhradu obdrzel nebo ma obdrzet platce za uskutecnene zdanitelne plneni od osoby, pro kterou plneni uskutecnil, nebo od treti osoby (§ 36 odst. 1).\n\nZaklad dane zahrnuje i vedlejsi vydaje (baleni, preprava, pojisteni) dle § 36 odst. 3.'
	},
	'sekce-kontrolni-hlaseni': {
		title: 'Sekce kontrolniho hlaseni (A4/A5/B2/B3)',
		simple:
			'Kontrolni hlaseni se deli na sekce podle smeru a velikosti plneni:\n\n- A4: Vydane faktury nad 10 000 Kc vcetne DPH (s detailem o odberateli)\n- A5: Vydane faktury do 10 000 Kc (souhrnne, bez detailu)\n- B2: Prijate faktury nad 10 000 Kc vcetne DPH (s detailem o dodavateli)\n- B3: Prijate faktury do 10 000 Kc (souhrnne, bez detailu)\n\nU A4 a B2 se uvadi DIC partnera, cislo dokladu a dalsi udaje.',
		legal:
			'Cleneni kontrolniho hlaseni stanovuje § 101c-101d zakona c. 235/2004 Sb. a pokyn GFR-D-57.\n\nOddil A obsahuje udaje o uskutecnenych plnenich (vystupy): A4 = plneni nad 10 000 Kc s identifikaci odberatele, A5 = ostatni plneni. Oddil B obsahuje udaje o prijatych plnenich (vstupy): B2 = plneni nad 10 000 Kc s identifikaci dodavatele, B3 = ostatni plneni.\n\nRozhodujici castka 10 000 Kc je vcetne DPH.'
	},
	dppd: {
		title: 'Datum poskytnuti danoveho plneni (DPPD)',
		simple:
			'DPPD je datum, ktere se uvadi v kontrolnim hlaseni. Odpovida datu uskutecneni plneni (DUZP) z faktury.\n\nPozor: DPPD neni datum vystaveni faktury ani datum splatnosti -- je to den, kdy skutecne doslo k dodani zbozi nebo poskyteni sluzby.',
		legal:
			'DPPD (datum poskytnuti/prijeti plneni) se uvadi v kontrolnim hlaseni dle § 101c zakona c. 235/2004 Sb. Odpovida datu uskutecneni zdanitelneho plneni (DUZP) dle § 21 tehoz zakona.\n\nV oddilech A4 a B2 kontrolniho hlaseni se DPPD uvadi u kazdeho radku. V oddilech A5 a B3 se neuvadi (plneni jsou agregovana).'
	},
	'kod-plneni': {
		title: 'Kod plneni',
		simple:
			'Kod plneni v souhrnnem hlaseni urcuje typ obchodu s partnerem v EU:\n\n- 0: Dodani zbozi do jine clenske zeme\n- 1: Poskytnuti sluzby podle § 9 odst. 1 (misto plneni u prijemce)\n- 2: Obchod v ramci triangulace (treti strana)\n- 3: Poskytnuti sluzby podle § 54 (financni a pojistovaci sluzby)',
		legal:
			'Kody plneni jsou definovany v § 102 zakona c. 235/2004 Sb. a v pokynu GFR k vyplnovani souhrnneho hlaseni.\n\nKod 0: dodani zbozi osobe registrovane k DPH v jinem clenskem state (§ 102 odst. 1 pism. a). Kod 1: poskytnuti sluzby s mistem plneni dle § 9 odst. 1 (§ 102 odst. 1 pism. d). Kod 2: dodani zbozi v ramci zjednoduseneho postupu pri tristrannnem obchodu (§ 102 odst. 1 pism. c). Kod 3: poskytnuti sluzby dle § 54.'
	},
	'zdanovaci-obdobi': {
		title: 'Zdanovaci obdobi',
		simple:
			'Zdanovaci obdobi je casovy usek, za ktery podavate DPH priznani a odvadite dan. Muze byt:\n\n- Mesicni: priznani podavate kazdy mesic (povinne pri obratu nad 10 mil. Kc)\n- Ctvrtletni: priznani podavate za kazde ctvrtleti (pro mensi platce DPH)\n\nPriznani se vzdy podava do 25. dne po skonceni obdobi.',
		legal:
			'Zdanovaci obdobi upravuji § 99-99a zakona c. 235/2004 Sb. Zakladnim zdanovacim obdobim je kalendarni mesic (§ 99). Platce muze zvolit ctvrtletni obdobi, pokud jeho obrat za predchazejici kalendarni rok nepresahl 10 000 000 Kc a neni nespolehlyvym platcem (§ 99a).\n\nZmena zdanovaciho obdobi se oznamuje spravci dane do konce ledna prislusneho roku (§ 99a odst. 2).'
	},
	'typ-faktury': {
		title: 'Typ dokladu',
		simple:
			'Faktura je danovy doklad, ktery vystavujete za dodane zbozi nebo sluzby. Zalohova faktura (proforma) je vyzva k platbe -- neni danovym dokladem a neslouzi k uplatneni DPH.\n\nPokud jste platce DPH, po uhrade zalohove faktury musite vystavit radnou fakturu (vyrovnani zalohy).',
		legal:
			'Danovy doklad je definovan v § 26 zakona c. 235/2004 Sb. o DPH. Zalohova faktura neni danovym dokladem ve smyslu tohoto zakona -- jedna se o obchodni dokument vyzyvajici k platbe.\n\nPovinne nalezitosti danoveho dokladu upravuje § 29 tehoz zakona. Po prijeti uhrady zalohove faktury vznika povinnost vystavit radny danovy doklad dle § 28 odst. 2.'
	},
	'dobropis': {
		title: 'Dobropis (opravny danovy doklad)',
		simple:
			'Dobropis je opravny doklad, ktery vystavujete, kdyz potrebujete snizit castku na jiz vydane fakture. Typicke duvody: sleva, reklamace, chybne uctovana castka nebo vraceni zbozi.\n\nDobropis odkazuje na puvodni fakturu a obsahuje zapornou castku. Po jeho vystaveni se snizi vase danove zavazky.',
		legal:
			'Opravny danovy doklad upravuje § 42 zakona c. 235/2004 Sb. o DPH. Platce je povinen vystavit opravny danovy doklad do 15 dni ode dne zjisteni skutecnosti rozhodnych pro provedeni opravy (§ 42 odst. 2).\n\nOpravny doklad musi obsahovat duvod opravy, rozdil mezi puvodni a novou castkou a odkaz na puvodni danovy doklad (§ 45 odst. 1).'
	},
	'vyrovnani-zalohy': {
		title: 'Vyrovnani zalohy',
		simple:
			'Po zaplaceni zalohove faktury (proformy) je treba vystavit radnou fakturu. Tato faktura obsahuje celkovou castku za dodane zbozi ci sluzby, od ktere se odecte jiz uhrazena zaloha.\n\nVysledkem je doplatek, ktery zakaznik jeste uhradi, nebo nulova castka, pokud zaloha pokryla vse.',
		legal:
			'Povinnost vystavit danovy doklad po prijeti uhrady vyplyva z § 21 odst. 1 zakona c. 235/2004 Sb. o DPH. Dnem prijeti uhrady vznika povinnost priznat dan na vystupu.\n\nPri vyrovnani se na radne fakture uvede celkova castka plneni a odecte se drive uhrazena zaloha. Zaklad dane a DPH se vypoctou z celkove castky plneni.'
	},
	'isdoc-export': {
		title: 'Export ISDOC',
		simple:
			'ISDOC je cesky standard pro elektronickou fakturaci. Soubor ve formatu ISDOC (.isdoc) obsahuje vsechna data faktury ve strojove citelne podobe.\n\nKdyz poslete fakturu ve formatu ISDOC, odberateluv ucetni system ji muze automaticky nacist bez rucniho prepisovani.',
		legal:
			'ISDOC (Information System Document) je cesky narodni standard elektronicke fakturace definovany ICT Unii. Format je zalozeny na UN/CEFACT a je kompatibilni s evropskou normou EN 16931.\n\nPouzivani elektronickych faktur upravuje § 26 odst. 3 a § 34 zakona c. 235/2004 Sb. Elektronicka faktura musi byt opatrena zarucenymi prostredky pro overeni puvodu a neporusenosti obsahu.'
	},
	'danova-kontrola': {
		title: 'Danova kontrola nakladu',
		simple:
			'Danova kontrola nakladu je proces, kdy systematicky projdete sve vydaje a overite, ze kazdy naklad je spravne dolozen, spravne zarazen a danove uznatelny.\n\nOznacenim nakladu jako "zkontrolovany" si udrzujete prehled o tom, ktere vydaje jste jiz overili a ktere jeste cekaji na kontrolu.',
		legal:
			'Danove uznatelne naklady jsou definovany v § 24-25 zakona c. 586/1992 Sb. o danich z prijmu. Podnikatel je povinen prokazat, ze vydaj byl vynalozen na dosazeni, zajisteni a udrzeni zdanitelnych prijmu.\n\nSpravce dane muze v ramci danove kontroly (§ 85 zakona c. 280/2009 Sb.) pozadovat prokazani opravnenosti vsech uplatnenych nakladu. Pravidelna kontrola minimalizuje riziko doplaceni dane.'
	},
	'ocr-import': {
		title: 'Import z dokladu (OCR)',
		simple:
			'OCR (opticke rozpoznavani znaku) automaticky precte text z nahrane faktury nebo uctenky. Staci nahrat soubor (PDF, JPG, PNG nebo WebP) a system se pokusi rozpoznat dodavatele, castku, datum a dalsi udaje.\n\nRozpoznana data muzete pred ulozenim zkontrolovat a upravit.',
		legal:
			'Archivace danovych dokladu v elektronicke podobe je upravena v § 35a zakona c. 235/2004 Sb. a § 31-32 zakona c. 563/1991 Sb. o ucetnictvi. Elektronicka kopie musi zachovat vernost a citelnost puvodniho dokladu.\n\nPovinnost uchovat danove doklady je 10 let od konce zdanovaciho obdobi (§ 35 zakona c. 235/2004 Sb.).'
	},
	'platebni-podminky': {
		title: 'Platebni podminky',
		simple:
			'Splatnost ve dnech urcuje, kolik dni od vystaveni faktury ma zakaznik na zaplaceni. Tato hodnota se automaticky nastavi na novych fakturach pro tohoto zakaznika.\n\nBezna splatnost je 14 nebo 30 dni. Pro stale zakazniky muzete nastavit individualni splatnost.',
		legal:
			'Splatnost je smluvni ujednani dle § 1958-1964 zakona c. 89/2012 Sb. (obcansky zakonik). Pro obchodni vztahy mezi podnikateli je maximalni smluvni splatnost 60 dni dle § 1963a OZ.\n\nPro vztahy s verejnym sektorem plati maximalni splatnost 30 dni (§ 1963 OZ). Delsi splatnost je mozna jen pokud to neni vuci veriteli hrube nespravedlive.'
	},
	'email-sablony': {
		title: 'Sablony emailu',
		simple:
			'Sablona emailu urcuje predmet a text zpravy, ktera se odesle zakaznikovi spolu s fakturou. Pouzijte {invoice_number} a system automaticky vlozi cislo faktury.\n\nSablonu nastavite jednou a pak se pouzije pro vsechny odeslane faktury. Pred odeslanim muzete text jeste upravit.',
		legal:
			'Odeslani faktury emailem je beznou obchodni praxi. Elektronicke doruceni danoveho dokladu je upraveno v § 34 zakona c. 235/2004 Sb. -- odberatel musi s elektronickym dorucenim souhlasit.\n\nElektronicka faktura musi splnovat podminky pro overeni puvodu a neporusenosti obsahu (§ 34 odst. 1).'
	},
	'opakovane-faktury': {
		title: 'Opakovane faktury',
		simple:
			'Opakovane faktury jsou sablony, ze kterych se automaticky generuji nove faktury v pravidelnych intervalech (mesicne, ctvrtletne, rocne).\n\nHodi se pro pausalni sluzby, najem, predplatne nebo jakoukoli pravidelnou fakturaci. Sablona obsahuje zakaznika, polozky a frekvenci -- system pak sam vytvori fakturu kdyz prisel cas.',
		legal:
			'Opakovane plneni je upraveno v § 21 odst. 8 zakona c. 235/2004 Sb. o DPH. U opakujiciho se plneni se DUZP stanovi nejpozdeji poslednim dnem zdanovaciho obdobi.\n\nSmlouvy na opakovane plneni (najem, servisni smlouvy) se ridi ustanovenimi o zavazkovem pravu v obcanskem zakoniku (§ 1724 a nasl. zakona c. 89/2012 Sb.).'
	},
	'kategorie-nakladu': {
		title: 'Kategorie nakladu',
		simple:
			'Kategorie pomahaji tridit naklady podle typu (kancelar, cestovne, sluzby, material apod.). Dobre roztridene naklady usnadnuji prehled o vydajich, pripravu danoveho priznani a komunikaci s ucetnim.\n\nMuzete pouzit vychozi kategorie nebo si vytvorit vlastni.',
		legal:
			'Trideni nakladu podle kategorii neni zakonem predepsano, ale vyplyva z povinnosti vest ucetnictvi prehledne a prukaze (§ 8 zakona c. 563/1991 Sb.).\n\nPro ucely danoveho priznani je vhodne clenit naklady dle § 24 zakona c. 586/1992 Sb. (danove uznatelne) a § 25 (neuznatelne), prip. dle povahy vydaje pro spravne vyplneni priloh priznani.'
	},
	'duplikace-faktury': {
		title: 'Duplikace faktury',
		simple:
			'Duplikace vytvori novou fakturu jako kopii stavajici. Zkopiruje se zakaznik, polozky, zpusob platby a dalsi nastaveni. Nova faktura dostane nove cislo a aktualni datumy.\n\nHodi se, kdyz vystavujete podobnou fakturu jako minule -- nemusite vse vyplnovat znovu.',
		legal:
			'Duplikovana faktura je novy, samostatny danovy doklad s vlastnim poradovym cislem dle § 29 zakona c. 235/2004 Sb. Jedna se o zcela nezavisly doklad, nikoliv o kopii puvodniho.\n\nPoradove cislo musi byt unikatni v ramci ciselne rady (§ 29 odst. 1 pism. b).'
	},
	'rocni-dane': {
		title: 'Rocni dane a prehledy OSVC',
		simple:
			'Rocni danove priznani (DPFO) a prehledy pro socialni (CSSZ) a zdravotni pojistovnu (ZP). Aplikace spocita zaklad dane z faktur a nakladu, aplikuje sazby a slevy, a vygeneruje XML pro elektronicke podani.',
		legal:
			'Danove priznani k dani z prijmu fyzickych osob (§ 38g zakona c. 586/1992 Sb.). Prehled o prijmech a vydajich OSVC pro CSSZ (§ 15 zakona c. 589/1992 Sb.) a pro zdravotni pojistovnu (§ 24 zakona c. 592/1992 Sb.).'
	}
};
