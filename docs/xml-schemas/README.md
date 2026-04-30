# XML Schema specifikace pro elektronická podání

Lokální kopie XSD schémat pro všechny formáty XML, které ZFaktury generuje
pro EPO (Finanční správa) a ČSSZ. Stažené z oficiálních zdrojů — slouží jako
referenční dokumentace, aby se nemusely při debugování validačních chyb znovu
hledat.

## Stav stažení

| Datum stažení | Zdroj |
|----|----|
| 2026-04-30 | https://adisspr.mfcr.cz/adis/jepo/schema/ (EPO) |
| 2026-04-30 | https://www.cssz.cz/definice-e-podani-osvc (ČSSZ) |

## EPO — Finanční správa

XML pro daňová podání přes portál MOJE daně (EPO). Validace probíhá při
nahrání souboru. Kořenový element všech podání je `<Pisemnost>`.

| Forma | Kód | XSD soubor | Verze v kódu | Generuje |
|----|----|----|----|----|
| Daň z příjmů FO | `DPFDP7` | [`epo/dpfdp7_epo2.xsd`](epo/dpfdp7_epo2.xsd) | `01.01.02` | `internal/annualtaxxml/income_tax_gen.go` |
| DPH přiznání | `DPHDP3` | [`epo/dphdp3_epo2.xsd`](epo/dphdp3_epo2.xsd) | `01.02.16` | `internal/vatxml/vat_return_gen.go` |
| Kontrolní hlášení DPH | `DPHKH1` | [`epo/dphkh1_epo2.xsd`](epo/dphkh1_epo2.xsd) | (řízeno schématem) | `internal/vatxml/control_statement_gen.go` |
| Souhrnné hlášení (VIES) | `DPHSHV` | [`epo/dphshv_epo2.xsd`](epo/dphshv_epo2.xsd) | (řízeno schématem) | `internal/vatxml/vies_gen.go` |

### Aktualizace XSD souborů

```bash
cd docs/xml-schemas/epo
for f in dpfdp7_epo2 dphdp3_epo2 dphkh1_epo2 dphshv_epo2; do
  curl -sfL "https://adisspr.mfcr.cz/adis/jepo/schema/${f}.xsd" -o "${f}.xsd"
done
```

EPO vydává nové verze schémat zpravidla 1× ročně (před začátkem zúčtovacího
období). Po stažení porovnej diff a aktualizuj `verzePis` v generátoru, pokud
se změnil — atribut je v XSD typu `xs:string`, takže neplatná verze projde
strukturální validací, ale EPO ji odmítne kontrolou.

### Číselníky používané v EPO

EPO validuje řadu atributů proti uzavřeným číselníkům (`c_ufo_cil`, `c_nace`,
`k_stat`, `k_uladis`, kódy obcí, …). Číselníky **nejsou** součástí XSD — XSD
jen omezuje typ (např. `\d{3,4}`). Kontrolu provádí EPO server až po nahrání.

| Číselník | Popis | Změna |
|----|----|----|
| `c_ufo_cil` | 3-místný kód krajského FÚ (451 Praha … 591 SFÚ) | stabilní od 2013 |
| `c_pracufo` | 4-místný kód územního pracoviště | stabilní od 2013 |
| CZ-NACE / OKEC | Klasifikace ekonomických činností | **NACE Rev. 2.1 od 1.1.2026** |
| `k_uladis` | Druh daně (`DPF`, `DPH`, …) | stabilní |
| `k_stat` | Kód státu (ISO 3166-1 alpha-2) | stabilní |

Reference:
- [Číselník ÚFO platný od 1.1.2013 — popis](https://podpora.mojedane.gov.cz/cs/seznam-okruhu/rozhrani-pro-treti-strany/informace-k-ciselniku-ufo-platnem-od-1-1-4382)
- [Změna CZ-NACE od 1.1.2026 (Finanční správa)](https://financnisprava.gov.cz/cs/financni-sprava/novinky/novinky-2026/zmena-ciselniku-nace-od-1-1-2026)
- [CZ-NACE 2025 — ČSÚ](https://csu.gov.cz/klasifikace-ekonomickych-cinnosti-cz-nace-platna-od-1-1-2025)

## ČSSZ — sociální pojištění

XML pro Přehled o příjmech a výdajích OSVČ (POSV) podávaný přes ePortál ČSSZ
nebo datovou schránku `5ffu6xk`. Schéma se mění každý rok (rok je součástí
`targetNamespace`).

| Forma | Rok | XSD soubor | Generuje |
|----|----|----|----|
| Přehled OSVČ 2025 | 2025 | [`cssz/OSVC25.xsd`](cssz/OSVC25.xsd) | `internal/annualtaxxml/social_insurance_gen.go` |
| Přehled OSVČ 2024 | 2024 | [`cssz/OSVC24.xsd`](cssz/OSVC24.xsd) | (historický) |
| baseTypes v2 | — | [`cssz/baseTypes2.xsd`](cssz/baseTypes2.xsd) | sdílené datové typy (od 1.6.2023) |

Namespace 2025: `http://schemas.cssz.cz/OSVC2025` — kód generátoru
(`Xmlns: "http://schemas.cssz.cz/OSVC2025"`) se musí každoročně aktualizovat
spolu s přidáním nového XSD.

### Aktualizace XSD souborů

URL ČSSZ obsahují interní GUID, takže je nelze odvodit z čísla roku — je
nutné je hledat na stránce
[Definice e-Podání OSVČ](https://www.cssz.cz/definice-e-podani-osvc).
Stažené pomocí `curl -sfL "<URL>" -o "OSVC<YY>.xsd"`.

## Diagnostika validačních chyb

### EPO control codes

EPO vrací při neúspěšné validaci číslo kontroly (např. `1671`) + textový
popis. Nejde o XSD chybu, ale o sémantickou kontrolu nad daty (číselníky,
formuláky, vzájemné dopočty řádků). Strukturální validace XSD probíhá *před*
sémantickou — pokud zafunguje XSD, číslo kontroly se neobjeví.

Časté chyby a jejich příčiny:

| Hláška | Pole | Typická příčina |
|----|----|----|
| `Číslo cílového finančního úřadu (xxxx) není v číselníku` | `c_ufo_cil` | 4-místný kód `c_pracufo` místo 3-místného `c_ufo` |
| `Kód CZ-NACE (xxxx) není uveden v číselníku` (control 1671) | `c_nace` | starý kód NACE Rev. 2 (pre-2026) místo NACE Rev. 2.1 |
| `kc_dazvyhod neodpovídá výpočtu` | `kc_dazvyhod` | součet `m_deti*` × měsíční sazba neodpovídá `kc_dazvyhod` |

### ČSSZ chyby

ČSSZ ePortál vrací XSD-úroveň validace s konkrétním XPath. Schéma je striktní
(`elementFormDefault="qualified"`), takže chybějící namespace `xmlns="..."`
na kořenu je nejčastější chyba.
