package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/fakturoid"
	"github.com/zajca/zfaktury/internal/repository"
)

// FakturoidClient defines the interface for fetching data from Fakturoid.
type FakturoidClient interface {
	ListSubjects(ctx context.Context) ([]fakturoid.Subject, error)
	ListInvoices(ctx context.Context) ([]fakturoid.Invoice, error)
	ListExpenses(ctx context.Context) ([]fakturoid.Expense, error)
}

// FakturoidImportService handles importing data from Fakturoid.
type FakturoidImportService struct {
	importRepo  repository.FakturoidImportLogRepo
	contactRepo repository.ContactRepo
	invoiceRepo repository.InvoiceRepo
	expenseRepo repository.ExpenseRepo
	contactSvc  *ContactService
	invoiceSvc  *InvoiceService
	expenseSvc  *ExpenseService
}

// NewFakturoidImportService creates a new FakturoidImportService.
func NewFakturoidImportService(
	importRepo repository.FakturoidImportLogRepo,
	contactRepo repository.ContactRepo,
	invoiceRepo repository.InvoiceRepo,
	expenseRepo repository.ExpenseRepo,
	contactSvc *ContactService,
	invoiceSvc *InvoiceService,
	expenseSvc *ExpenseService,
) *FakturoidImportService {
	return &FakturoidImportService{
		importRepo:  importRepo,
		contactRepo: contactRepo,
		invoiceRepo: invoiceRepo,
		expenseRepo: expenseRepo,
		contactSvc:  contactSvc,
		invoiceSvc:  invoiceSvc,
		expenseSvc:  expenseSvc,
	}
}

// ImportAll fetches all data from Fakturoid and imports new entities, skipping duplicates.
func (s *FakturoidImportService) ImportAll(ctx context.Context, client FakturoidClient) (*domain.FakturoidImportResult, error) {
	subjects, err := client.ListSubjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching Fakturoid subjects: %w", err)
	}

	invoices, err := client.ListInvoices(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching Fakturoid invoices: %w", err)
	}

	expenses, err := client.ListExpenses(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching Fakturoid expenses: %w", err)
	}

	preview := s.buildPreview(ctx, subjects, invoices, expenses)

	result := &domain.FakturoidImportResult{}

	// Import contacts first (invoices/expenses depend on them)
	subjectMap := make(map[int64]int64) // fakturoid subject_id -> local contact_id

	for _, item := range preview.Contacts {
		if item.Status == "duplicate" && item.ExistingID != nil {
			subjectMap[item.FakturoidID] = *item.ExistingID
			result.ContactsSkipped++
			continue
		}
		if item.Status != "new" {
			result.ContactsSkipped++
			continue
		}

		contact := item.Entity.(*domain.Contact)
		if err := s.contactSvc.Create(ctx, contact); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("contact %d: %v", item.FakturoidID, err))
			continue
		}
		subjectMap[item.FakturoidID] = contact.ID

		if err := s.importRepo.Create(ctx, &domain.FakturoidImportLog{
			FakturoidEntityType: "subject",
			FakturoidID:         item.FakturoidID,
			LocalEntityType:     "contact",
			LocalID:             contact.ID,
		}); err != nil {
			slog.Warn("failed to log import", "entity", "contact", "fakturoid_id", item.FakturoidID, "error", err)
		}
		result.ContactsCreated++
	}

	// Import invoices
	for _, item := range preview.Invoices {
		if item.Status != "new" {
			result.InvoicesSkipped++
			continue
		}

		invoice := item.Entity.(*domain.Invoice)
		if invoice.CustomerID == 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("invoice %d: customer not resolved", item.FakturoidID))
			continue
		}
		if localID, ok := subjectMap[invoice.CustomerID]; ok {
			invoice.CustomerID = localID
		}

		if err := s.invoiceSvc.Create(ctx, invoice); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("invoice %d: %v", item.FakturoidID, err))
			continue
		}

		if invoice.Status != domain.InvoiceStatusDraft {
			_ = s.invoiceRepo.UpdateStatus(ctx, invoice.ID, invoice.Status)
		}

		if err := s.importRepo.Create(ctx, &domain.FakturoidImportLog{
			FakturoidEntityType: "invoice",
			FakturoidID:         item.FakturoidID,
			LocalEntityType:     "invoice",
			LocalID:             invoice.ID,
		}); err != nil {
			slog.Warn("failed to log import", "entity", "invoice", "fakturoid_id", item.FakturoidID, "error", err)
		}
		result.InvoicesCreated++
	}

	// Import expenses
	for _, item := range preview.Expenses {
		if item.Status != "new" {
			result.ExpensesSkipped++
			continue
		}

		expense := item.Entity.(*domain.Expense)
		if expense.VendorID != nil {
			fakturoidVendorID := *expense.VendorID
			if localID, ok := subjectMap[fakturoidVendorID]; ok {
				expense.VendorID = &localID
			}
		}

		if err := s.expenseSvc.Create(ctx, expense); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("expense %d: %v", item.FakturoidID, err))
			continue
		}

		if err := s.importRepo.Create(ctx, &domain.FakturoidImportLog{
			FakturoidEntityType: "expense",
			FakturoidID:         item.FakturoidID,
			LocalEntityType:     "expense",
			LocalID:             expense.ID,
		}); err != nil {
			slog.Warn("failed to log import", "entity", "expense", "fakturoid_id", item.FakturoidID, "error", err)
		}
		result.ExpensesCreated++
	}

	return result, nil
}

// buildPreview constructs a preview from already-fetched Fakturoid data.
func (s *FakturoidImportService) buildPreview(ctx context.Context, subjects []fakturoid.Subject, invoices []fakturoid.Invoice, expenses []fakturoid.Expense) *domain.FakturoidImportPreview {
	preview := &domain.FakturoidImportPreview{}

	// Build subject ID -> local contact ID map for resolving references
	subjectMap := make(map[int64]int64) // fakturoid subject_id -> local contact_id

	// Process contacts
	for _, subj := range subjects {
		item := s.previewContact(ctx, subj)
		if item.Status == "duplicate" && item.ExistingID != nil {
			subjectMap[subj.ID] = *item.ExistingID
		}
		preview.Contacts = append(preview.Contacts, item)
	}

	// Process invoices
	for _, inv := range invoices {
		item := s.previewInvoice(ctx, inv, subjectMap)
		preview.Invoices = append(preview.Invoices, item)
	}

	// Process expenses
	for _, exp := range expenses {
		item := s.previewExpense(ctx, exp, subjectMap)
		preview.Expenses = append(preview.Expenses, item)
	}

	return preview
}

// previewContact maps a Fakturoid subject to a domain contact and checks for duplicates.
func (s *FakturoidImportService) previewContact(ctx context.Context, subj fakturoid.Subject) domain.FakturoidImportItem {
	contact := mapSubjectToContact(subj)

	// Check import log first
	logEntry, err := s.importRepo.FindByFakturoidID(ctx, "subject", subj.ID)
	if err != nil {
		slog.Warn("import log lookup failed", "entity", "subject", "id", subj.ID, "error", err)
	}
	if logEntry != nil {
		return domain.FakturoidImportItem{
			FakturoidID: subj.ID,
			Status:      "duplicate",
			Entity:      contact,
			ExistingID:  &logEntry.LocalID,
			Reason:      "Already imported",
		}
	}

	// Check by ICO
	if contact.ICO != "" {
		existing, err := s.contactRepo.FindByICO(ctx, contact.ICO)
		if err == nil && existing != nil {
			return domain.FakturoidImportItem{
				FakturoidID: subj.ID,
				Status:      "duplicate",
				Entity:      contact,
				ExistingID:  &existing.ID,
				Reason:      fmt.Sprintf("Existing contact with ICO %s", contact.ICO),
			}
		}
	}

	// Check by exact name
	contacts, _, err := s.contactRepo.List(ctx, domain.ContactFilter{Search: contact.Name, Limit: 1})
	if err == nil && len(contacts) > 0 && contacts[0].Name == contact.Name {
		existingID := contacts[0].ID
		return domain.FakturoidImportItem{
			FakturoidID: subj.ID,
			Status:      "duplicate",
			Entity:      contact,
			ExistingID:  &existingID,
			Reason:      fmt.Sprintf("Existing contact with name %q", contact.Name),
		}
	}

	return domain.FakturoidImportItem{
		FakturoidID: subj.ID,
		Status:      "new",
		Entity:      contact,
	}
}

// previewInvoice maps a Fakturoid invoice and checks for duplicates.
func (s *FakturoidImportService) previewInvoice(ctx context.Context, inv fakturoid.Invoice, subjectMap map[int64]int64) domain.FakturoidImportItem {
	invoice := mapFakturoidInvoice(inv, subjectMap)

	// Check import log
	logEntry, err := s.importRepo.FindByFakturoidID(ctx, "invoice", inv.ID)
	if err != nil {
		slog.Warn("import log lookup failed", "entity", "invoice", "id", inv.ID, "error", err)
	}
	if logEntry != nil {
		return domain.FakturoidImportItem{
			FakturoidID: inv.ID,
			Status:      "duplicate",
			Entity:      invoice,
			ExistingID:  &logEntry.LocalID,
			Reason:      "Already imported",
		}
	}

	// Check by invoice number
	if invoice.InvoiceNumber != "" {
		invoices, _, err := s.invoiceRepo.List(ctx, domain.InvoiceFilter{Search: invoice.InvoiceNumber, Limit: 1})
		if err == nil && len(invoices) > 0 && invoices[0].InvoiceNumber == invoice.InvoiceNumber {
			existingID := invoices[0].ID
			return domain.FakturoidImportItem{
				FakturoidID: inv.ID,
				Status:      "duplicate",
				Entity:      invoice,
				ExistingID:  &existingID,
				Reason:      fmt.Sprintf("Existing invoice %s", invoice.InvoiceNumber),
			}
		}
	}

	return domain.FakturoidImportItem{
		FakturoidID: inv.ID,
		Status:      "new",
		Entity:      invoice,
	}
}

// previewExpense maps a Fakturoid expense and checks for duplicates.
func (s *FakturoidImportService) previewExpense(ctx context.Context, exp fakturoid.Expense, subjectMap map[int64]int64) domain.FakturoidImportItem {
	expense := mapFakturoidExpense(exp, subjectMap)

	// Check import log
	logEntry, err := s.importRepo.FindByFakturoidID(ctx, "expense", exp.ID)
	if err != nil {
		slog.Warn("import log lookup failed", "entity", "expense", "id", exp.ID, "error", err)
	}
	if logEntry != nil {
		return domain.FakturoidImportItem{
			FakturoidID: exp.ID,
			Status:      "duplicate",
			Entity:      expense,
			ExistingID:  &logEntry.LocalID,
			Reason:      "Already imported",
		}
	}

	// Check by expense number + vendor + date
	if expense.ExpenseNumber != "" {
		issueDate := expense.IssueDate
		expFilter := domain.ExpenseFilter{
			Search:   expense.ExpenseNumber,
			VendorID: expense.VendorID,
			DateFrom: &issueDate,
			DateTo:   &issueDate,
			Limit:    1,
		}
		expenses, _, err := s.expenseRepo.List(ctx, expFilter)
		if err == nil && len(expenses) > 0 && expenses[0].ExpenseNumber == expense.ExpenseNumber {
			existingID := expenses[0].ID
			return domain.FakturoidImportItem{
				FakturoidID: exp.ID,
				Status:      "duplicate",
				Entity:      expense,
				ExistingID:  &existingID,
				Reason:      fmt.Sprintf("Existing expense %s", expense.ExpenseNumber),
			}
		}
	}

	return domain.FakturoidImportItem{
		FakturoidID: exp.ID,
		Status:      "new",
		Entity:      expense,
	}
}

// mapSubjectToContact converts a Fakturoid Subject to a domain Contact.
func mapSubjectToContact(subj fakturoid.Subject) *domain.Contact {
	contact := &domain.Contact{
		Name:             subj.Name,
		ICO:              subj.RegistrationNo,
		DIC:              subj.VatNo,
		Street:           subj.Street,
		City:             subj.City,
		ZIP:              subj.Zip,
		Country:          subj.Country,
		IBAN:             subj.IBAN,
		Email:            subj.Email,
		Phone:            subj.Phone,
		Web:              subj.Web,
		Type:             domain.ContactTypeCompany,
		PaymentTermsDays: subj.Due,
	}

	// Parse Czech bank account format "cislo/kod"
	if subj.BankAccount != "" {
		parts := strings.SplitN(subj.BankAccount, "/", 2)
		contact.BankAccount = parts[0]
		if len(parts) == 2 {
			contact.BankCode = parts[1]
		}
	}

	return contact
}

// mapFakturoidInvoice converts a Fakturoid Invoice to a domain Invoice.
func mapFakturoidInvoice(inv fakturoid.Invoice, subjectMap map[int64]int64) *domain.Invoice {
	invoice := &domain.Invoice{
		InvoiceNumber:  inv.Number,
		VariableSymbol: inv.VariableSymbol,
		CurrencyCode:   inv.Currency,
		ExchangeRate:   domain.FromFloat(inv.ExchangeRate),
		SubtotalAmount: domain.FromFloat(inv.Subtotal),
		TotalAmount:    domain.FromFloat(inv.Total),
		Notes:          inv.Note,
	}

	// Map document type
	switch inv.DocumentType {
	case "proforma":
		invoice.Type = domain.InvoiceTypeProforma
	case "credit_note":
		invoice.Type = domain.InvoiceTypeCreditNote
	default:
		invoice.Type = domain.InvoiceTypeRegular
	}

	// Map status
	switch inv.Status {
	case "paid":
		invoice.Status = domain.InvoiceStatusPaid
	case "overdue":
		invoice.Status = domain.InvoiceStatusOverdue
	case "cancelled":
		invoice.Status = domain.InvoiceStatusCancelled
	default:
		invoice.Status = domain.InvoiceStatusSent
	}

	// Parse dates
	if t, err := time.Parse("2006-01-02", inv.IssuedOn); err == nil {
		invoice.IssueDate = t
	}
	if t, err := time.Parse("2006-01-02", inv.DueOn); err == nil {
		invoice.DueDate = t
	}
	if t, err := time.Parse("2006-01-02", inv.TaxableFulfillmentDue); err == nil {
		invoice.DeliveryDate = t
	}

	// Set paid date from first payment
	if len(inv.Payments) > 0 && inv.Payments[0].PaidOn != "" {
		if t, err := time.Parse("2006-01-02", inv.Payments[0].PaidOn); err == nil {
			invoice.PaidAt = &t
			invoice.PaidAmount = invoice.TotalAmount
		}
	}

	// Resolve customer
	if localID, ok := subjectMap[inv.SubjectID]; ok {
		invoice.CustomerID = localID
	} else {
		// Store fakturoid subject ID temporarily for resolution during import
		invoice.CustomerID = inv.SubjectID
	}

	// Map line items
	for i, line := range inv.Lines {
		invoice.Items = append(invoice.Items, domain.InvoiceItem{
			Description:    line.Name,
			Quantity:       domain.FromFloat(line.Quantity),
			Unit:           line.UnitName,
			UnitPrice:      domain.FromFloat(line.UnitPrice),
			VATRatePercent: int(line.VatRate),
			SortOrder:      i + 1,
		})
	}

	// Calculate VAT from items
	invoice.CalculateTotals()

	return invoice
}

// mapFakturoidExpense converts a Fakturoid Expense to a domain Expense.
func mapFakturoidExpense(exp fakturoid.Expense, subjectMap map[int64]int64) *domain.Expense {
	expense := &domain.Expense{
		ExpenseNumber:   exp.OriginalNumber,
		Amount:          domain.FromFloat(exp.Total),
		CurrencyCode:    exp.Currency,
		ExchangeRate:    domain.FromFloat(exp.ExchangeRate),
		IsTaxDeductible: true,
		BusinessPercent: 100,
		Notes:           exp.PrivateNote,
	}

	// Description: prefer description field, fallback to first line name
	expense.Description = exp.Description
	if expense.Description == "" && len(exp.Lines) > 0 {
		expense.Description = exp.Lines[0].Name
	}
	if expense.Description == "" {
		expense.Description = "Import z Fakturoidu"
	}

	// Parse issue date
	if t, err := time.Parse("2006-01-02", exp.IssuedOn); err == nil {
		expense.IssueDate = t
	}

	// Map payment method
	switch exp.PaymentMethod {
	case "cash":
		expense.PaymentMethod = "cash"
	default:
		expense.PaymentMethod = "bank_transfer"
	}

	// Calculate dominant VAT rate and VAT amount from lines
	if len(exp.Lines) > 0 {
		vatRateCounts := make(map[int]float64)
		var totalVAT float64
		for _, line := range exp.Lines {
			rate := int(line.VatRate)
			lineTotal := line.Quantity * line.UnitPrice
			vatRateCounts[rate] += lineTotal
			if rate > 0 {
				totalVAT += lineTotal * line.VatRate / (100 + line.VatRate)
			}
		}
		// Find dominant rate
		var maxAmount float64
		for rate, amount := range vatRateCounts {
			if amount > maxAmount {
				maxAmount = amount
				expense.VATRatePercent = rate
			}
		}
		expense.VATAmount = domain.FromFloat(totalVAT)
	}

	// Resolve vendor
	if exp.SubjectID > 0 {
		if localID, ok := subjectMap[exp.SubjectID]; ok {
			expense.VendorID = &localID
		} else {
			vendorID := exp.SubjectID
			expense.VendorID = &vendorID
		}
	}

	return expense
}
