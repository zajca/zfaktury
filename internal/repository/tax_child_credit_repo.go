package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// TaxChildCreditRepository handles persistence of TaxChildCredit entities.
type TaxChildCreditRepository struct {
	db *sql.DB
}

// NewTaxChildCreditRepository creates a new TaxChildCreditRepository.
func NewTaxChildCreditRepository(db *sql.DB) *TaxChildCreditRepository {
	return &TaxChildCreditRepository{db: db}
}

// taxChildCreditColumns is the list of columns to select for a TaxChildCredit row.
const taxChildCreditColumns = `id, year, child_name, birth_number, child_order, months_claimed, ztp, credit_amount, created_at, updated_at`

// scanTaxChildCredit scans a TaxChildCredit from a row.
func scanTaxChildCredit(s scanner) (*domain.TaxChildCredit, error) {
	c := &domain.TaxChildCredit{}
	var createdAtStr, updatedAtStr string
	var ztp int

	err := s.Scan(
		&c.ID, &c.Year, &c.ChildName, &c.BirthNumber,
		&c.ChildOrder, &c.MonthsClaimed, &ztp, &c.CreditAmount,
		&createdAtStr, &updatedAtStr,
	)
	if err != nil {
		return nil, err
	}

	c.ZTP = ztp != 0

	c.CreatedAt, err = parseDate(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning tax_child_credit created_at: %w", err)
	}
	c.UpdatedAt, err = parseDate(time.RFC3339, updatedAtStr)
	if err != nil {
		return nil, fmt.Errorf("scanning tax_child_credit updated_at: %w", err)
	}

	return c, nil
}

// Create inserts a new child credit into the database.
func (r *TaxChildCreditRepository) Create(ctx context.Context, credit *domain.TaxChildCredit) error {
	now := time.Now()
	credit.CreatedAt = now
	credit.UpdatedAt = now

	var ztp int
	if credit.ZTP {
		ztp = 1
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO tax_child_credits (year, child_name, birth_number, child_order, months_claimed, ztp, credit_amount, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		credit.Year, credit.ChildName, credit.BirthNumber,
		credit.ChildOrder, credit.MonthsClaimed, ztp, credit.CreditAmount,
		credit.CreatedAt.Format(time.RFC3339), credit.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("inserting tax_child_credit: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id for tax_child_credit: %w", err)
	}
	credit.ID = id
	return nil
}

// Update modifies an existing child credit.
func (r *TaxChildCreditRepository) Update(ctx context.Context, credit *domain.TaxChildCredit) error {
	credit.UpdatedAt = time.Now()

	var ztp int
	if credit.ZTP {
		ztp = 1
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE tax_child_credits SET
			year = ?, child_name = ?, birth_number = ?, child_order = ?,
			months_claimed = ?, ztp = ?, credit_amount = ?, updated_at = ?
		WHERE id = ?`,
		credit.Year, credit.ChildName, credit.BirthNumber, credit.ChildOrder,
		credit.MonthsClaimed, ztp, credit.CreditAmount,
		credit.UpdatedAt.Format(time.RFC3339), credit.ID,
	)
	if err != nil {
		return fmt.Errorf("updating tax_child_credit %d: %w", credit.ID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for tax_child_credit %d: %w", credit.ID, err)
	}
	if rows == 0 {
		return fmt.Errorf("tax_child_credit %d: %w", credit.ID, domain.ErrNotFound)
	}
	return nil
}

// Delete removes a child credit by ID.
func (r *TaxChildCreditRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM tax_child_credits WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting tax_child_credit %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected for tax_child_credit %d delete: %w", id, err)
	}
	if rows == 0 {
		return fmt.Errorf("tax_child_credit %d: %w", id, domain.ErrNotFound)
	}
	return nil
}

// ListByYear retrieves all child credits for a given year, ordered by child_order.
func (r *TaxChildCreditRepository) ListByYear(ctx context.Context, year int) ([]domain.TaxChildCredit, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+taxChildCreditColumns+` FROM tax_child_credits WHERE year = ? ORDER BY child_order`,
		year,
	)
	if err != nil {
		return nil, fmt.Errorf("listing tax_child_credits for year %d: %w", year, err)
	}
	defer func() { _ = rows.Close() }()

	var result []domain.TaxChildCredit
	for rows.Next() {
		credit, err := scanTaxChildCredit(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning tax_child_credit row: %w", err)
		}
		result = append(result, *credit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tax_child_credit rows: %w", err)
	}
	return result, nil
}
