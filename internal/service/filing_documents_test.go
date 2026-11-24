package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

var filedAtDelivery = time.Date(2026, 11, 24, 12, 7, 0, 0, time.UTC)

// markFiledAtCase drives MarkFiledAt of one document service: create a draft,
// mark it filed at a past time, then mark it again.
type markFiledAtCase struct {
	create     func(t *testing.T) int64
	markFiled  func(ctx context.Context, id int64, at time.Time) error
	markNow    func(ctx context.Context, id int64) error
	filedAt    func(ctx context.Context, id int64) (*time.Time, string)
	reMarkable bool // MarkFiled (without a time) of a filed document is not an error
	doc        FilingDocument
}

func TestMarkFiledAt_UsesTheGivenTimeAndReportsAFiledDocument(t *testing.T) {
	ctx := context.Background()

	cases := map[string]func(t *testing.T) markFiledAtCase{
		"vat return": func(t *testing.T) markFiledAtCase {
			svc, _ := setupVATReturnSvc(t)
			return markFiledAtCase{
				doc: NewFilingDocuments(svc, nil, nil, nil, nil, nil)[domain.FilingKindVATReturn],
				create: func(t *testing.T) int64 {
					vr := &domain.VATReturn{Period: domain.TaxPeriod{Year: 2025, Month: 3}, FilingType: domain.FilingTypeRegular}
					if err := svc.Create(ctx, vr); err != nil {
						t.Fatal(err)
					}
					return vr.ID
				},
				markFiled: func(ctx context.Context, id int64, at time.Time) error {
					_, err := svc.MarkFiledAt(ctx, id, at)
					return err
				},
				markNow: func(ctx context.Context, id int64) error { _, err := svc.MarkFiled(ctx, id); return err },
				filedAt: func(ctx context.Context, id int64) (*time.Time, string) {
					vr, _ := svc.repo.GetByID(ctx, id)
					return vr.FiledAt, vr.Status
				},
			}
		},
		"control statement": func(t *testing.T) markFiledAtCase {
			svc, _ := setupVATControlSvc(t)
			return markFiledAtCase{
				doc: NewFilingDocuments(nil, svc, nil, nil, nil, nil)[domain.FilingKindControlStatement],
				create: func(t *testing.T) int64 {
					cs := &domain.VATControlStatement{Period: domain.TaxPeriod{Year: 2025, Month: 3}, FilingType: domain.FilingTypeRegular}
					if err := svc.Create(ctx, cs); err != nil {
						t.Fatal(err)
					}
					return cs.ID
				},
				markFiled: svc.MarkFiledAt,
				markNow:   svc.MarkFiled,
				filedAt: func(ctx context.Context, id int64) (*time.Time, string) {
					cs, _ := svc.repo.GetByID(ctx, id)
					return cs.FiledAt, cs.Status
				},
				reMarkable: true,
			}
		},
		"VIES summary": func(t *testing.T) markFiledAtCase {
			svc, _ := newVIESSvc(t)
			return markFiledAtCase{
				doc: NewFilingDocuments(nil, nil, svc, nil, nil, nil)[domain.FilingKindVIESSummary],
				create: func(t *testing.T) int64 {
					vs := validVIESSummary()
					if err := svc.Create(ctx, vs); err != nil {
						t.Fatal(err)
					}
					return vs.ID
				},
				markFiled: svc.MarkFiledAt,
				markNow:   svc.MarkFiled,
				filedAt: func(ctx context.Context, id int64) (*time.Time, string) {
					vs, _ := svc.repo.GetByID(ctx, id)
					return vs.FiledAt, vs.Status
				},
			}
		},
		"income tax return": func(t *testing.T) markFiledAtCase {
			svc, _ := setupIncomeTaxSvc(t)
			return markFiledAtCase{
				doc: NewFilingDocuments(nil, nil, nil, svc, nil, nil)[domain.FilingKindIncomeTax],
				create: func(t *testing.T) int64 {
					itr := &domain.IncomeTaxReturn{Year: 2025, FilingType: domain.FilingTypeRegular}
					if err := svc.Create(ctx, itr); err != nil {
						t.Fatal(err)
					}
					return itr.ID
				},
				markFiled: func(ctx context.Context, id int64, at time.Time) error {
					_, err := svc.MarkFiledAt(ctx, id, at)
					return err
				},
				markNow: func(ctx context.Context, id int64) error { _, err := svc.MarkFiled(ctx, id); return err },
				filedAt: func(ctx context.Context, id int64) (*time.Time, string) {
					itr, _ := svc.repo.GetByID(ctx, id)
					return itr.FiledAt, itr.Status
				},
			}
		},
		"social insurance": func(t *testing.T) markFiledAtCase {
			svc, _ := setupSocialInsuranceSvc(t)
			return markFiledAtCase{
				doc: NewFilingDocuments(nil, nil, nil, nil, svc, nil)[domain.FilingKindSocialInsurance],
				create: func(t *testing.T) int64 {
					sio := &domain.SocialInsuranceOverview{Year: 2025, FilingType: domain.FilingTypeRegular, ActivityMonths: domain.FullYearActivityMonths}
					if err := svc.Create(ctx, sio); err != nil {
						t.Fatal(err)
					}
					return sio.ID
				},
				markFiled: func(ctx context.Context, id int64, at time.Time) error {
					_, err := svc.MarkFiledAt(ctx, id, at)
					return err
				},
				markNow: func(ctx context.Context, id int64) error { _, err := svc.MarkFiled(ctx, id); return err },
				filedAt: func(ctx context.Context, id int64) (*time.Time, string) {
					sio, _ := svc.repo.GetByID(ctx, id)
					return sio.FiledAt, sio.Status
				},
			}
		},
	}

	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			c := build(t)
			id := c.create(t)

			if filed, err := c.doc.IsFiled(ctx, id); err != nil || filed {
				t.Fatalf("IsFiled of a draft = %v, %v, want false", filed, err)
			}
			if _, err := c.doc.IsFiled(ctx, id+1000); err == nil {
				t.Error("IsFiled of a missing document returned no error")
			}
			if err := c.markFiled(ctx, id, filedAtDelivery); err != nil {
				t.Fatalf("MarkFiledAt: %v", err)
			}
			at, status := c.filedAt(ctx, id)
			if status != domain.FilingStatusFiled || at == nil || !at.Equal(filedAtDelivery) {
				t.Fatalf("filed state = %q at %v, want filed at %v", status, at, filedAtDelivery)
			}

			if filed, err := c.doc.IsFiled(ctx, id); err != nil || !filed {
				t.Errorf("IsFiled after MarkFiledAt = %v, %v, want true", filed, err)
			}

			// A second mark reports ErrFilingAlreadyFiled and keeps the first time.
			err := c.markFiled(ctx, id, filedAtDelivery.Add(24*time.Hour))
			if !errors.Is(err, domain.ErrFilingAlreadyFiled) {
				t.Fatalf("second MarkFiledAt = %v, want ErrFilingAlreadyFiled", err)
			}
			if again, _ := c.filedAt(ctx, id); !again.Equal(filedAtDelivery) {
				t.Errorf("FiledAt = %v after the refused mark, want %v", again, filedAtDelivery)
			}
		})
	}
}

func TestMarkFiled_WithoutATimeKeepsItsBehaviour(t *testing.T) {
	ctx := context.Background()

	vies, _ := newVIESSvc(t)
	vs := validVIESSummary()
	if err := vies.Create(ctx, vs); err != nil {
		t.Fatal(err)
	}
	before := time.Now().Add(-time.Second)
	if err := vies.MarkFiled(ctx, vs.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := vies.repo.GetByID(ctx, vs.ID)
	if got.FiledAt == nil || got.FiledAt.Before(before) {
		t.Errorf("MarkFiled set FiledAt = %v, want about now", got.FiledAt)
	}
	// MarkFiled keeps reporting an invalid input, not the new sentinel.
	err := vies.MarkFiled(ctx, vs.ID)
	if !errors.Is(err, domain.ErrInvalidInput) || errors.Is(err, domain.ErrFilingAlreadyFiled) {
		t.Errorf("VIES MarkFiled twice = %v, want ErrInvalidInput only", err)
	}

	control, _ := setupVATControlSvc(t)
	cs := &domain.VATControlStatement{Period: domain.TaxPeriod{Year: 2025, Month: 3}, FilingType: domain.FilingTypeRegular}
	if err := control.Create(ctx, cs); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := control.MarkFiled(ctx, cs.ID); err != nil {
			t.Errorf("control statement MarkFiled #%d = %v, want no error as before", i+1, err)
		}
	}
}
