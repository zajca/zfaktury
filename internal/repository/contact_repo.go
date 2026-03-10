package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// ContactRepository handles persistence of Contact entities.
type ContactRepository struct {
	db *sql.DB
}

// NewContactRepository creates a new ContactRepository.
func NewContactRepository(db *sql.DB) *ContactRepository {
	return &ContactRepository{db: db}
}

// Create inserts a new contact into the database.
func (r *ContactRepository) Create(ctx context.Context, c *domain.Contact) error {
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO contacts (
			type, name, ico, dic, street, city, zip, country,
			email, phone, web, bank_account, bank_code, iban, swift,
			payment_terms_days, tags, notes, is_favorite, vat_unreliable_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Type, c.Name, c.ICO, c.DIC, c.Street, c.City, c.ZIP, c.Country,
		c.Email, c.Phone, c.Web, c.BankAccount, c.BankCode, c.IBAN, c.SWIFT,
		c.PaymentTermsDays, c.Tags, c.Notes, c.IsFavorite, c.VATUnreliableAt,
		c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting contact: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for contact: %w", err)
	}
	c.ID = id
	return nil
}

// Update modifies an existing contact.
func (r *ContactRepository) Update(ctx context.Context, c *domain.Contact) error {
	c.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		UPDATE contacts SET
			type = ?, name = ?, ico = ?, dic = ?, street = ?, city = ?, zip = ?, country = ?,
			email = ?, phone = ?, web = ?, bank_account = ?, bank_code = ?, iban = ?, swift = ?,
			payment_terms_days = ?, tags = ?, notes = ?, is_favorite = ?, vat_unreliable_at = ?,
			updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`,
		c.Type, c.Name, c.ICO, c.DIC, c.Street, c.City, c.ZIP, c.Country,
		c.Email, c.Phone, c.Web, c.BankAccount, c.BankCode, c.IBAN, c.SWIFT,
		c.PaymentTermsDays, c.Tags, c.Notes, c.IsFavorite, c.VATUnreliableAt,
		c.UpdatedAt, c.ID,
	)
	if err != nil {
		return fmt.Errorf("updating contact %d: %w", c.ID, err)
	}
	return nil
}

// Delete performs a soft delete on a contact.
func (r *ContactRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		UPDATE contacts SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now, now, id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting contact %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for contact %d: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("contact %d not found or already deleted", id)
	}
	return nil
}

// GetByID retrieves a single contact by ID.
func (r *ContactRepository) GetByID(ctx context.Context, id int64) (*domain.Contact, error) {
	c := &domain.Contact{}
	var createdAtStr string
	var updatedAtStr string
	var deletedAtStr sql.NullString
	var vatUnreliableAtStr sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, type, name, ico, dic, street, city, zip, country,
			email, phone, web, bank_account, bank_code, iban, swift,
			payment_terms_days, tags, notes, is_favorite, vat_unreliable_at,
			created_at, updated_at, deleted_at
		FROM contacts WHERE id = ? AND deleted_at IS NULL`, id,
	).Scan(
		&c.ID, &c.Type, &c.Name, &c.ICO, &c.DIC, &c.Street, &c.City, &c.ZIP, &c.Country,
		&c.Email, &c.Phone, &c.Web, &c.BankAccount, &c.BankCode, &c.IBAN, &c.SWIFT,
		&c.PaymentTermsDays, &c.Tags, &c.Notes, &c.IsFavorite, &vatUnreliableAtStr,
		&createdAtStr, &updatedAtStr, &deletedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("contact %d not found: %w", id, err)
		}
		return nil, fmt.Errorf("querying contact %d: %w", id, err)
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	if deletedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, deletedAtStr.String)
		c.DeletedAt = &t
	}
	if vatUnreliableAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, vatUnreliableAtStr.String)
		c.VATUnreliableAt = &t
	}
	return c, nil
}

// List retrieves contacts matching the given filter with pagination.
// Returns the matching contacts, total count, and any error.
func (r *ContactRepository) List(ctx context.Context, filter domain.ContactFilter) ([]domain.Contact, int, error) {
	where := "deleted_at IS NULL"
	args := []any{}

	if filter.Search != "" {
		where += " AND (name LIKE ? OR ico LIKE ? OR email LIKE ?)"
		search := "%" + filter.Search + "%"
		args = append(args, search, search, search)
	}
	if filter.Type != "" {
		where += " AND type = ?"
		args = append(args, filter.Type)
	}
	if filter.Favorite != nil {
		where += " AND is_favorite = ?"
		args = append(args, *filter.Favorite)
	}

	// Count total matching rows.
	var total int
	countQuery := "SELECT COUNT(*) FROM contacts WHERE " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting contacts: %w", err)
	}

	// Fetch page.
	query := "SELECT id, type, name, ico, dic, street, city, zip, country, " +
		"email, phone, web, bank_account, bank_code, iban, swift, " +
		"payment_terms_days, tags, notes, is_favorite, vat_unreliable_at, " +
		"created_at, updated_at, deleted_at " +
		"FROM contacts WHERE " + where + " ORDER BY name ASC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d OFFSET %d", filter.Limit, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing contacts: %w", err)
	}
	defer rows.Close()

	var contacts []domain.Contact
	for rows.Next() {
		var c domain.Contact
		var createdAtStr string
		var updatedAtStr string
		var deletedAtStr sql.NullString
		var vatUnreliableAtStr sql.NullString
		if err := rows.Scan(
			&c.ID, &c.Type, &c.Name, &c.ICO, &c.DIC, &c.Street, &c.City, &c.ZIP, &c.Country,
			&c.Email, &c.Phone, &c.Web, &c.BankAccount, &c.BankCode, &c.IBAN, &c.SWIFT,
			&c.PaymentTermsDays, &c.Tags, &c.Notes, &c.IsFavorite, &vatUnreliableAtStr,
			&createdAtStr, &updatedAtStr, &deletedAtStr,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning contact row: %w", err)
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		if deletedAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, deletedAtStr.String)
			c.DeletedAt = &t
		}
		if vatUnreliableAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, vatUnreliableAtStr.String)
			c.VATUnreliableAt = &t
		}
		contacts = append(contacts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating contact rows: %w", err)
	}
	return contacts, total, nil
}

// FindByICO finds a contact by its Czech business identification number (ICO).
func (r *ContactRepository) FindByICO(ctx context.Context, ico string) (*domain.Contact, error) {
	c := &domain.Contact{}
	var createdAtStr string
	var updatedAtStr string
	var deletedAtStr sql.NullString
	var vatUnreliableAtStr sql.NullString

	err := r.db.QueryRowContext(ctx, `
		SELECT id, type, name, ico, dic, street, city, zip, country,
			email, phone, web, bank_account, bank_code, iban, swift,
			payment_terms_days, tags, notes, is_favorite, vat_unreliable_at,
			created_at, updated_at, deleted_at
		FROM contacts WHERE ico = ? AND deleted_at IS NULL`, ico,
	).Scan(
		&c.ID, &c.Type, &c.Name, &c.ICO, &c.DIC, &c.Street, &c.City, &c.ZIP, &c.Country,
		&c.Email, &c.Phone, &c.Web, &c.BankAccount, &c.BankCode, &c.IBAN, &c.SWIFT,
		&c.PaymentTermsDays, &c.Tags, &c.Notes, &c.IsFavorite, &vatUnreliableAtStr,
		&createdAtStr, &updatedAtStr, &deletedAtStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("contact with ICO %s not found: %w", ico, err)
		}
		return nil, fmt.Errorf("querying contact by ICO %s: %w", ico, err)
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	if deletedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, deletedAtStr.String)
		c.DeletedAt = &t
	}
	if vatUnreliableAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, vatUnreliableAtStr.String)
		c.VATUnreliableAt = &t
	}
	return c, nil
}
