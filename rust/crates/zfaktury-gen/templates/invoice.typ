// Invoice PDF template for ZFaktury.
// Data is passed from Rust via sys.inputs.

#import sys: inputs

// Page setup: A4 with 15mm margins.
#set page(paper: "a4", margin: 15mm)
#set text(font: "Liberation Sans", size: 9pt, lang: "cs")

// --- Helper functions ---

#let accent = rgb(inputs.accent_color)
#let accent_light = accent.lighten(85%)

#let label_style(body) = text(size: 8pt, weight: "bold", body)
#let value_style(body) = text(size: 8pt, body)
#let small_style(body) = text(size: 7pt, body)

// --- Header: Invoice type + number ---

#grid(
  columns: (1fr, auto),
  align: (left, right),
  text(size: 16pt, weight: "bold", fill: accent)[#inputs.type_label #inputs.invoice_number],
  text(size: 10pt, weight: "bold")[#inputs.status_label],
)

#v(4pt)

// --- Dates row ---

#grid(
  columns: (1fr, 1fr, 1fr),
  align: (left, left, right),
  small_style[Datum vystavení: #inputs.issue_date],
  small_style[Datum splatnosti: #inputs.due_date],
  small_style[DÚZP: #inputs.delivery_date],
)

#v(4pt)
#line(length: 100%, stroke: 0.5pt + luma(180))
#v(6pt)

// --- Parties: Supplier (left) | Customer (right) ---

#grid(
  columns: (1fr, 1fr),
  gutter: 16pt,
  [
    #text(size: 10pt, weight: "bold")[Dodavatel]
    #v(2pt)
    #text(weight: "bold")[#inputs.supplier_name] \
    #if inputs.supplier_street != "" [#inputs.supplier_street \ ]
    #if inputs.supplier_zip != "" or inputs.supplier_city != "" [#inputs.supplier_zip #inputs.supplier_city \ ]
    #if inputs.supplier_ico != "" [IČO: #inputs.supplier_ico \ ]
    #if inputs.supplier_dic != "" [DIČ: #inputs.supplier_dic \ ]
    #if inputs.supplier_email != "" [#small_style[#inputs.supplier_email] \ ]
    #if inputs.supplier_phone != "" [#small_style[#inputs.supplier_phone]]
  ],
  [
    #text(size: 10pt, weight: "bold")[Odběratel]
    #v(2pt)
    #if inputs.has_customer == "true" [
      #text(weight: "bold")[#inputs.customer_name] \
      #if inputs.customer_street != "" [#inputs.customer_street \ ]
      #if inputs.customer_zip != "" or inputs.customer_city != "" [#inputs.customer_zip #inputs.customer_city \ ]
      #if inputs.customer_ico != "" [IČO: #inputs.customer_ico \ ]
      #if inputs.customer_dic != "" [DIČ: #inputs.customer_dic \ ]
    ]
  ],
)

#v(6pt)
#line(length: 100%, stroke: 0.5pt + luma(180))
#v(6pt)

// --- Items table ---

#table(
  columns: (auto, 2fr, auto, auto, 1fr, auto, 1fr, 1fr),
  align: (center, left, right, left, right, right, right, right),
  stroke: none,
  fill: (x, y) => if y == 0 { accent_light } else { none },
  table.hline(y: 1, stroke: 0.5pt + luma(150)),
  table.header(
    text(size: 7pt, weight: "bold")[\#],
    text(size: 7pt, weight: "bold")[Popis],
    text(size: 7pt, weight: "bold")[Mn.],
    text(size: 7pt, weight: "bold")[Jedn.],
    text(size: 7pt, weight: "bold")[Cena/ks],
    text(size: 7pt, weight: "bold")[DPH %],
    text(size: 7pt, weight: "bold")[DPH],
    text(size: 7pt, weight: "bold")[Celkem],
  ),
  ..inputs.items.map(item => (
    value_style(item.index),
    value_style(item.description),
    value_style(item.quantity),
    value_style(item.unit),
    value_style(item.unit_price),
    value_style(item.vat_rate),
    value_style(item.vat_amount),
    value_style(item.total),
  )).flatten(),
)

#v(4pt)

// --- VAT summary ---

#if inputs.vat_groups.len() > 0 [
  #align(right)[
    #table(
      columns: (auto, auto, auto),
      align: (left, right, right),
      stroke: none,
      table.header(
        text(size: 7pt, weight: "bold")[Sazba DPH],
        text(size: 7pt, weight: "bold")[Základ],
        text(size: 7pt, weight: "bold")[DPH],
      ),
      table.hline(stroke: 0.3pt + luma(180)),
      ..inputs.vat_groups.map(g => (
        small_style(g.rate),
        small_style(g.base),
        small_style(g.vat),
      )).flatten(),
    )
  ]
]

#v(4pt)

// --- Totals ---

#align(right)[
  #grid(
    columns: (auto, auto),
    column-gutter: 12pt,
    row-gutter: 4pt,
    align: (right, right),
    text(size: 9pt)[Základ celkem:], text(size: 9pt)[#inputs.subtotal_czk],
    text(size: 9pt)[DPH celkem:], text(size: 9pt)[#inputs.vat_czk],
    text(size: 11pt, weight: "bold")[Celkem k úhradě:], text(size: 11pt, weight: "bold")[#inputs.total_czk],
  )
]

#v(6pt)
#line(length: 100%, stroke: 0.5pt + luma(180))
#v(6pt)

// --- Payment section ---

#text(size: 10pt, weight: "bold")[Platební údaje]
#v(4pt)

#if inputs.show_bank_details == "true" [
  #if inputs.bank_account_full != "" [
    Číslo účtu: #inputs.bank_account_full \
  ]
  #if inputs.iban != "" [
    IBAN: #inputs.iban \
  ]
  #if inputs.swift != "" [
    SWIFT: #inputs.swift \
  ]
]

#if inputs.variable_symbol != "" [
  Variabilní symbol: #inputs.variable_symbol \
]
#if inputs.constant_symbol != "" [
  Konstantní symbol: #inputs.constant_symbol \
]

#text(weight: "bold")[Datum splatnosti: #inputs.due_date]

// --- QR code ---

#if inputs.has_qr == "true" [
  #v(6pt)
  #text(size: 8pt, weight: "bold")[QR platba:]
  #v(2pt)
  #image.decode(bytes(inputs.qr_png), width: 3cm)
]

// --- Footer ---

#v(1fr)

#if inputs.vat_note != "" [
  #align(center)[
    #text(size: 7pt, style: "italic")[#inputs.vat_note]
  ]
]

#if inputs.footer_text != "" [
  #align(center)[
    #text(size: 7pt)[#inputs.footer_text]
  ]
]
