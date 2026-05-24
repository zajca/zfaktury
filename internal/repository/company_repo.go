package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// CompanyRepository persists Company aggregates.
//
// Unlike per-company repositories, CompanyRepository operates without
// a companyID filter — it knows about all companies regardless of which
// is currently active.
type CompanyRepository struct {
	db *sql.DB
}

// NewCompanyRepository constructs a CompanyRepository backed by the given DB.
func NewCompanyRepository(db *sql.DB) *CompanyRepository {
	return &CompanyRepository{db: db}
}

const companyColumns = `id, name, legal_name, ico, dic, vat_registered,
	street, house_number, city, zip,
	email, phone,
	first_name, last_name,
	bank_account, bank_code, iban, swift,
	logo_path, accent_color,
	created_at, updated_at, deleted_at`

// scanCompany maps a single row (from either *sql.Row or *sql.Rows) into a domain.Company.
func scanCompany(row interface{ Scan(...any) error }) (domain.Company, error) {
	var c domain.Company
	var vatInt int
	var dic, street, hn, city, zip, email, phone sql.NullString
	var firstName, lastName, bankAcc, bankCode, iban, swift sql.NullString
	var logo, accent sql.NullString
	var createdAt, updatedAt string
	var deletedAt sql.NullString

	err := row.Scan(
		&c.ID, &c.Name, &c.LegalName, &c.ICO, &dic, &vatInt,
		&street, &hn, &city, &zip,
		&email, &phone,
		&firstName, &lastName,
		&bankAcc, &bankCode, &iban, &swift,
		&logo, &accent,
		&createdAt, &updatedAt, &deletedAt,
	)
	if err != nil {
		return c, err
	}
	c.DIC = dic.String
	c.VATRegistered = vatInt == 1
	c.Street, c.HouseNumber, c.City, c.ZIP = street.String, hn.String, city.String, zip.String
	c.Email, c.Phone = email.String, phone.String
	c.FirstName, c.LastName = firstName.String, lastName.String
	c.BankAccount, c.BankCode, c.IBAN, c.SWIFT = bankAcc.String, bankCode.String, iban.String, swift.String
	c.LogoPath, c.AccentColor = logo.String, accent.String
	c.CreatedAt, err = parseDate(time.RFC3339, createdAt)
	if err != nil {
		return c, fmt.Errorf("parsing created_at: %w", err)
	}
	c.UpdatedAt, err = parseDate(time.RFC3339, updatedAt)
	if err != nil {
		return c, fmt.Errorf("parsing updated_at: %w", err)
	}
	c.DeletedAt, err = parseDatePtr(time.RFC3339, deletedAt)
	if err != nil {
		return c, fmt.Errorf("parsing deleted_at: %w", err)
	}
	return c, nil
}

// Create inserts a new company and returns its generated ID.
func (r *CompanyRepository) Create(ctx context.Context, c domain.Company) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	vatInt := 0
	if c.VATRegistered {
		vatInt = 1
	}
	res, err := r.db.ExecContext(ctx, `
INSERT INTO companies (name, legal_name, ico, dic, vat_registered,
	street, house_number, city, zip,
	email, phone,
	first_name, last_name,
	bank_account, bank_code, iban, swift,
	logo_path, accent_color,
	created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Name, c.LegalName, c.ICO, nullableString(c.DIC), vatInt,
		nullableString(c.Street), nullableString(c.HouseNumber), nullableString(c.City), nullableString(c.ZIP),
		nullableString(c.Email), nullableString(c.Phone),
		nullableString(c.FirstName), nullableString(c.LastName),
		nullableString(c.BankAccount), nullableString(c.BankCode), nullableString(c.IBAN), nullableString(c.SWIFT),
		nullableString(c.LogoPath), nullableString(c.AccentColor),
		now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting company: %w", err)
	}
	return res.LastInsertId()
}

// GetByID fetches an active (non-soft-deleted) company by primary key.
func (r *CompanyRepository) GetByID(ctx context.Context, id int64) (domain.Company, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+companyColumns+` FROM companies WHERE id = ? AND deleted_at IS NULL`, id)
	c, err := scanCompany(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Company{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Company{}, fmt.Errorf("fetching company %d: %w", id, err)
	}
	return c, nil
}

// List returns all active companies ordered by ID.
func (r *CompanyRepository) List(ctx context.Context) ([]domain.Company, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+companyColumns+` FROM companies WHERE deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("listing companies: %w", err)
	}
	defer rows.Close()

	var out []domain.Company
	for rows.Next() {
		c, err := scanCompany(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning company row: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Update replaces the mutable fields of an existing company.
func (r *CompanyRepository) Update(ctx context.Context, c domain.Company) error {
	vatInt := 0
	if c.VATRegistered {
		vatInt = 1
	}
	res, err := r.db.ExecContext(ctx, `
UPDATE companies SET
	name = ?, legal_name = ?, ico = ?, dic = ?, vat_registered = ?,
	street = ?, house_number = ?, city = ?, zip = ?,
	email = ?, phone = ?,
	first_name = ?, last_name = ?,
	bank_account = ?, bank_code = ?, iban = ?, swift = ?,
	logo_path = ?, accent_color = ?,
	updated_at = ?
WHERE id = ? AND deleted_at IS NULL`,
		c.Name, c.LegalName, c.ICO, nullableString(c.DIC), vatInt,
		nullableString(c.Street), nullableString(c.HouseNumber), nullableString(c.City), nullableString(c.ZIP),
		nullableString(c.Email), nullableString(c.Phone),
		nullableString(c.FirstName), nullableString(c.LastName),
		nullableString(c.BankAccount), nullableString(c.BankCode), nullableString(c.IBAN), nullableString(c.SWIFT),
		nullableString(c.LogoPath), nullableString(c.AccentColor),
		time.Now().UTC().Format(time.RFC3339),
		c.ID,
	)
	if err != nil {
		return fmt.Errorf("updating company %d: %w", c.ID, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// SoftDelete marks a company as deleted by setting deleted_at.
func (r *CompanyRepository) SoftDelete(ctx context.Context, id int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx,
		`UPDATE companies SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at IS NULL`,
		now, now, id,
	)
	if err != nil {
		return fmt.Errorf("soft-deleting company %d: %w", id, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CountActive returns the number of non-soft-deleted companies.
func (r *CompanyRepository) CountActive(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM companies WHERE deleted_at IS NULL`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("counting active companies: %w", err)
	}
	return n, nil
}

// nullableString converts an empty string to a SQL NULL value, so blank
// optional fields stay distinguishable from explicit empty values.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
