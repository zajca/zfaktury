package email

import (
	"strings"
	"testing"
)

func TestReminderTemplate_Level1(t *testing.T) {
	data := ReminderData{
		CustomerName:   "Acme s.r.o.",
		InvoiceNumber:  "FV20260001",
		TotalAmount:    "12 345,67 Kc",
		DueDate:        "10. 3. 2026",
		DaysOverdue:    5,
		BankAccount:    "1234567890/0100",
		VariableSymbol: "20260001",
		UserName:       "Jan Novak",
	}

	subject, html, text := ReminderTemplate(1, data)

	if !strings.Contains(subject, "Připomenutí splatnosti") {
		t.Errorf("level 1 subject missing expected text: %q", subject)
	}
	if !strings.Contains(subject, data.InvoiceNumber) {
		t.Errorf("level 1 subject missing invoice number: %q", subject)
	}
	if !strings.Contains(text, "dovolujeme si Vás upozornit") {
		t.Errorf("level 1 text missing polite phrasing: %q", text)
	}
	if !strings.Contains(text, data.TotalAmount) {
		t.Errorf("level 1 text missing total amount")
	}
	if !strings.Contains(text, data.BankAccount) {
		t.Errorf("level 1 text missing bank account")
	}
	if !strings.Contains(text, data.VariableSymbol) {
		t.Errorf("level 1 text missing variable symbol")
	}
	if !strings.Contains(text, data.UserName) {
		t.Errorf("level 1 text missing user name")
	}
	if !strings.Contains(html, "<html>") {
		t.Errorf("level 1 html missing HTML tags")
	}
}

func TestReminderTemplate_Level2(t *testing.T) {
	data := ReminderData{
		CustomerName:   "Acme s.r.o.",
		InvoiceNumber:  "FV20260001",
		TotalAmount:    "12 345,67 Kc",
		DueDate:        "10. 3. 2026",
		DaysOverdue:    20,
		BankAccount:    "1234567890/0100",
		VariableSymbol: "20260001",
		UserName:       "Jan Novak",
	}

	subject, _, text := ReminderTemplate(2, data)

	if !strings.Contains(subject, "Druhá upomínka") {
		t.Errorf("level 2 subject missing expected text: %q", subject)
	}
	if !strings.Contains(text, "dosud jsme neobdrželi") {
		t.Errorf("level 2 text missing firm phrasing: %q", text)
	}
}

func TestReminderTemplate_Level3(t *testing.T) {
	data := ReminderData{
		CustomerName:   "Acme s.r.o.",
		InvoiceNumber:  "FV20260001",
		TotalAmount:    "12 345,67 Kc",
		DueDate:        "10. 3. 2026",
		DaysOverdue:    45,
		BankAccount:    "1234567890/0100",
		VariableSymbol: "20260001",
		UserName:       "Jan Novak",
	}

	subject, _, text := ReminderTemplate(3, data)

	if !strings.Contains(subject, "Poslední upomínka") {
		t.Errorf("level 3 subject missing expected text: %q", subject)
	}
	if !strings.Contains(text, "přes opakovanou upomínku") {
		t.Errorf("level 3 text missing urgent phrasing: %q", text)
	}
}

func TestReminderTemplate_ClampsLevel(t *testing.T) {
	data := ReminderData{
		InvoiceNumber: "FV001",
		UserName:      "Test",
	}

	// Level 0 should be treated as 1.
	subj0, _, _ := ReminderTemplate(0, data)
	subj1, _, _ := ReminderTemplate(1, data)
	if subj0 != subj1 {
		t.Errorf("level 0 should clamp to level 1, got subject: %q vs %q", subj0, subj1)
	}

	// Level 5 should be treated as 3.
	subj5, _, _ := ReminderTemplate(5, data)
	subj3, _, _ := ReminderTemplate(3, data)
	if subj5 != subj3 {
		t.Errorf("level 5 should clamp to level 3, got subject: %q vs %q", subj5, subj3)
	}
}
