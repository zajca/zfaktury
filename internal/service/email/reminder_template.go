package email

import (
	"fmt"
	"strings"
)

// ReminderData holds the data used to render reminder email templates.
type ReminderData struct {
	CustomerName   string
	InvoiceNumber  string
	TotalAmount    string // formatted, e.g. "12 345,67 Kč"
	DueDate        string // formatted, e.g. "10. 3. 2026"
	DaysOverdue    int
	BankAccount    string
	VariableSymbol string
	UserName       string // sender's name
}

// ReminderTemplate returns the subject, HTML body, and plain text body for a
// payment reminder at the given escalation level (1-3). Levels above 3 are
// treated as level 3.
func ReminderTemplate(level int, data ReminderData) (string, string, string) {
	if level < 1 {
		level = 1
	}
	if level > 3 {
		level = 3
	}

	switch level {
	case 1:
		return reminderLevel1(data)
	case 2:
		return reminderLevel2(data)
	default:
		return reminderLevel3(data)
	}
}

func reminderLevel1(d ReminderData) (string, string, string) {
	subject := fmt.Sprintf("Připomenutí splatnosti faktury %s", d.InvoiceNumber)

	text := fmt.Sprintf(`Vážený zákazníku,

dovolujeme si Vás upozornit, že faktura %s na částku %s se splatností %s nebyla dosud uhrazena. Faktura je %d dní po splatnosti.

Prosíme o uhrazení na náš bankovní účet %s pod variabilním symbolem %s.

Pokud jste platbu již odeslali, považujte prosím tuto zprávu za bezpředmětnou.

S pozdravem
%s`,
		d.InvoiceNumber, d.TotalAmount, d.DueDate, d.DaysOverdue,
		d.BankAccount, d.VariableSymbol, d.UserName)

	html := wrapHTML(subject, strings.ReplaceAll(escapeHTML(text), "\n", "<br>\n"))

	return subject, html, text
}

func reminderLevel2(d ReminderData) (string, string, string) {
	subject := fmt.Sprintf("Druhá upomínka - faktura %s po splatnosti", d.InvoiceNumber)

	text := fmt.Sprintf(`Vážený zákazníku,

dosud jsme neobdrželi platbu za fakturu %s na částku %s, která byla splatná dne %s. Faktura je nyní %d dní po splatnosti.

Žádáme Vás o neprodlené uhrazení na bankovní účet %s, variabilní symbol %s.

V případě dotazů nás prosím kontaktujte.

S pozdravem
%s`,
		d.InvoiceNumber, d.TotalAmount, d.DueDate, d.DaysOverdue,
		d.BankAccount, d.VariableSymbol, d.UserName)

	html := wrapHTML(subject, strings.ReplaceAll(escapeHTML(text), "\n", "<br>\n"))

	return subject, html, text
}

func reminderLevel3(d ReminderData) (string, string, string) {
	subject := fmt.Sprintf("Poslední upomínka - faktura %s", d.InvoiceNumber)

	text := fmt.Sprintf(`Vážený zákazníku,

přes opakovanou upomínku evidujeme neuhrazenou fakturu %s na částku %s se splatností %s. Faktura je %d dní po splatnosti.

Toto je poslední upomínka před předáním pohledávky k dalšímu řešení. Žádáme Vás o okamžité uhrazení na bankovní účet %s, variabilní symbol %s.

S pozdravem
%s`,
		d.InvoiceNumber, d.TotalAmount, d.DueDate, d.DaysOverdue,
		d.BankAccount, d.VariableSymbol, d.UserName)

	html := wrapHTML(subject, strings.ReplaceAll(escapeHTML(text), "\n", "<br>\n"))

	return subject, html, text
}

func wrapHTML(title, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>%s</title></head>
<body style="font-family: sans-serif; line-height: 1.6; color: #333;">
<p>%s</p>
</body>
</html>`, escapeHTML(title), body)
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
