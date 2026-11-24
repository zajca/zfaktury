package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zajca/zfaktury/internal/domain"
)

// filingDocument adapts one document service to FilingDocument.
type filingDocument struct {
	regenerate  func(ctx context.Context, id int64) ([]byte, error)
	markFiledAt func(ctx context.Context, id int64, at time.Time) error
	status      func(ctx context.Context, id int64) (string, error)
}

// IsFiled reports whether the document is already filed. It reads the stored
// status only and never regenerates anything.
func (d filingDocument) IsFiled(ctx context.Context, id int64) (bool, error) {
	status, err := d.status(ctx, id)
	if err != nil {
		return false, err
	}
	return status == domain.FilingStatusFiled, nil
}

func (d filingDocument) RegenerateXML(ctx context.Context, id int64) ([]byte, error) {
	return d.regenerate(ctx, id)
}

func (d filingDocument) MarkFiledAt(ctx context.Context, id int64, at time.Time) error {
	return d.markFiledAt(ctx, id, at)
}

// taxpayerDIC reads the DIC the control statement and VIES generators need.
func taxpayerDIC(ctx context.Context, settings FilingSettings) (string, error) {
	dic, err := settings.Get(ctx, SettingDIC)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("reading the DIC: %w", err)
	}
	dic = strings.TrimSpace(dic)
	if dic == "" {
		return "", fmt.Errorf("generating XML: %w: %w", domain.ErrMissingSetting,
			&domain.UserMessageError{Message: "V nastavení chybí DIČ."})
	}
	return dic, nil
}

// NewFilingDocuments adapts the document services for FilingService. Each
// RegenerateXML regenerates and stores the XML through the document's own
// service; MarkFiledAt marks the document filed at the given time.
func NewFilingDocuments(
	vatReturn *VATReturnService,
	control *VATControlStatementService,
	vies *VIESSummaryService,
	incomeTax *IncomeTaxReturnService,
	social *SocialInsuranceService,
	settings FilingSettings,
) map[domain.FilingKind]FilingDocument {
	return map[domain.FilingKind]FilingDocument{
		domain.FilingKindVATReturn: filingDocument{
			regenerate: func(ctx context.Context, id int64) ([]byte, error) {
				vr, err := vatReturn.GenerateXML(ctx, id)
				if err != nil {
					return nil, err
				}
				return vr.XMLData, nil
			},
			markFiledAt: func(ctx context.Context, id int64, at time.Time) error {
				_, err := vatReturn.MarkFiledAt(ctx, id, at)
				return err
			},
			status: func(ctx context.Context, id int64) (string, error) {
				vr, err := vatReturn.GetByID(ctx, id)
				if err != nil {
					return "", err
				}
				return vr.Status, nil
			},
		},
		domain.FilingKindControlStatement: filingDocument{
			regenerate: func(ctx context.Context, id int64) ([]byte, error) {
				dic, err := taxpayerDIC(ctx, settings)
				if err != nil {
					return nil, err
				}
				return control.GenerateXML(ctx, id, dic)
			},
			markFiledAt: control.MarkFiledAt,
			status: func(ctx context.Context, id int64) (string, error) {
				cs, err := control.GetByID(ctx, id)
				if err != nil {
					return "", err
				}
				return cs.Status, nil
			},
		},
		domain.FilingKindVIESSummary: filingDocument{
			regenerate: func(ctx context.Context, id int64) ([]byte, error) {
				dic, err := taxpayerDIC(ctx, settings)
				if err != nil {
					return nil, err
				}
				if err := vies.GenerateXML(ctx, id, dic); err != nil {
					return nil, err
				}
				return vies.GetXMLData(ctx, id)
			},
			markFiledAt: vies.MarkFiledAt,
			status: func(ctx context.Context, id int64) (string, error) {
				vs, err := vies.GetByID(ctx, id)
				if err != nil {
					return "", err
				}
				return vs.Status, nil
			},
		},
		domain.FilingKindIncomeTax: filingDocument{
			regenerate: func(ctx context.Context, id int64) ([]byte, error) {
				itr, err := incomeTax.GenerateXML(ctx, id)
				if err != nil {
					return nil, err
				}
				return itr.XMLData, nil
			},
			markFiledAt: func(ctx context.Context, id int64, at time.Time) error {
				_, err := incomeTax.MarkFiledAt(ctx, id, at)
				return err
			},
			status: func(ctx context.Context, id int64) (string, error) {
				itr, err := incomeTax.GetByID(ctx, id)
				if err != nil {
					return "", err
				}
				return itr.Status, nil
			},
		},
		domain.FilingKindSocialInsurance: filingDocument{
			regenerate: func(ctx context.Context, id int64) ([]byte, error) {
				sio, err := social.GenerateXML(ctx, id)
				if err != nil {
					return nil, err
				}
				return sio.XMLData, nil
			},
			markFiledAt: func(ctx context.Context, id int64, at time.Time) error {
				_, err := social.MarkFiledAt(ctx, id, at)
				return err
			},
			status: func(ctx context.Context, id int64) (string, error) {
				sio, err := social.GetByID(ctx, id)
				if err != nil {
					return "", err
				}
				return sio.Status, nil
			},
		},
	}
}
