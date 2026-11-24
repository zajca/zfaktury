package service

import (
	"context"
	"errors"
	"testing"

	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/epo"
)

type fakeEPOSource struct {
	data  []byte
	err   error
	gotID int64
}

func (f *fakeEPOSource) GetXMLData(_ context.Context, id int64) ([]byte, error) {
	f.gotID = id
	return f.data, f.err
}

type fakeEPOClient struct {
	findings    []epo.Finding
	checkErr    error
	link        string
	openErr     error
	checkCalls  int
	openCalls   int
	checkedWith []byte
	openedWith  []byte

	checkPackaging epo.Packaging
	openPackaging  epo.Packaging
}

func (f *fakeEPOClient) Check(_ context.Context, xmlData []byte, packaging epo.Packaging) ([]epo.Finding, error) {
	f.checkCalls++
	f.checkedWith = xmlData
	f.checkPackaging = packaging
	return f.findings, f.checkErr
}

func (f *fakeEPOClient) OpenForm(_ context.Context, xmlData []byte, packaging epo.Packaging) (string, error) {
	f.openCalls++
	f.openedWith = xmlData
	f.openPackaging = packaging
	return f.link, f.openErr
}

func newEPOServiceForTest(src *fakeEPOSource, client *fakeEPOClient) *EPOService {
	return NewEPOService(map[EPOKind]EPOXMLSource{EPOKindVATReturn: src}, client)
}

func TestEPOKind_Packaging(t *testing.T) {
	want := map[EPOKind]epo.Packaging{
		EPOKindControlStatement: epo.PackagingZIP,
		EPOKindVATReturn:        epo.PackagingPlain,
		EPOKindVIESSummary:      epo.PackagingPlain,
		EPOKindIncomeTax:        epo.PackagingPlain,
	}
	for kind, packaging := range want {
		if got := kind.Packaging(); got != packaging {
			t.Errorf("%s packaging = %d, want %d", kind, got, packaging)
		}
	}
}

// The kind decides the packaging of both the check and the form request.
func TestEPOService_OpenForm_RoutesPackagingByKind(t *testing.T) {
	for kind, want := range map[EPOKind]epo.Packaging{
		EPOKindControlStatement: epo.PackagingZIP,
		EPOKindVATReturn:        epo.PackagingPlain,
		EPOKindVIESSummary:      epo.PackagingPlain,
		EPOKindIncomeTax:        epo.PackagingPlain,
	} {
		t.Run(string(kind), func(t *testing.T) {
			src := &fakeEPOSource{data: []byte("<Pisemnost/>")}
			client := &fakeEPOClient{link: "https://mojedane.gov.cz/dpr/form?form_id=FAKE"}
			svc := NewEPOService(map[EPOKind]EPOXMLSource{kind: src}, client)

			if _, err := svc.OpenForm(context.Background(), kind, 1); err != nil {
				t.Fatalf("OpenForm: %v", err)
			}
			if client.checkPackaging != want || client.openPackaging != want {
				t.Errorf("packaging check/open = %d/%d, want %d/%d", client.checkPackaging, client.openPackaging, want, want)
			}
		})
	}
}

func TestEPOService_OpenForm_Success(t *testing.T) {
	src := &fakeEPOSource{data: []byte("<Pisemnost/>")}
	warn := epo.Finding{Type: epo.TypePassable, Code: "PODANO_PO_LHUTE", Field: "d_poddp"}
	client := &fakeEPOClient{findings: []epo.Finding{warn}, link: "https://mojedane.gov.cz/dpr/form?form_id=FAKE"}
	svc := newEPOServiceForTest(src, client)

	got, err := svc.OpenForm(context.Background(), EPOKindVATReturn, 7)
	if err != nil {
		t.Fatalf("OpenForm: %v", err)
	}
	if got.URL != client.link {
		t.Errorf("URL = %q, want %q", got.URL, client.link)
	}
	if len(got.Warnings) != 1 || got.Warnings[0] != warn {
		t.Errorf("warnings = %+v, want [%+v]", got.Warnings, warn)
	}
	if src.gotID != 7 {
		t.Errorf("source asked for id %d, want 7", src.gotID)
	}
	if client.checkCalls != 1 || client.openCalls != 1 {
		t.Errorf("calls check/open = %d/%d, want 1/1", client.checkCalls, client.openCalls)
	}
	if string(client.checkedWith) != "<Pisemnost/>" || string(client.openedWith) != "<Pisemnost/>" {
		t.Errorf("client did not receive the stored XML")
	}
}

func TestEPOService_OpenForm_EmptyXML(t *testing.T) {
	for name, data := range map[string][]byte{"nil": nil, "empty": {}} {
		t.Run(name, func(t *testing.T) {
			client := &fakeEPOClient{}
			svc := newEPOServiceForTest(&fakeEPOSource{data: data}, client)

			_, err := svc.OpenForm(context.Background(), EPOKindVATReturn, 1)
			if !errors.Is(err, domain.ErrXMLNotGenerated) {
				t.Fatalf("err = %v, want ErrXMLNotGenerated", err)
			}
			if client.checkCalls != 0 || client.openCalls != 0 {
				t.Errorf("EPO must not be called without XML")
			}
		})
	}
}

func TestEPOService_OpenForm_BlockingFindingStopsBeforeOpen(t *testing.T) {
	for _, typ := range []string{epo.TypeCritical, epo.TypeStructure, epo.TypeException} {
		t.Run(typ, func(t *testing.T) {
			findings := []epo.Finding{
				{Type: epo.TypePassable, Code: "WARN"},
				{Type: typ, Code: "BLOCK", Field: "dic"},
			}
			client := &fakeEPOClient{findings: findings, link: "https://mojedane.gov.cz/dpr/x"}
			svc := newEPOServiceForTest(&fakeEPOSource{data: []byte("<Pisemnost/>")}, client)

			_, err := svc.OpenForm(context.Background(), EPOKindVATReturn, 1)
			var rejected *epo.RejectedError
			if !errors.As(err, &rejected) {
				t.Fatalf("err = %v, want *epo.RejectedError", err)
			}
			if len(rejected.Findings) != 2 {
				t.Errorf("rejected findings = %d, want all 2", len(rejected.Findings))
			}
			if client.openCalls != 0 {
				t.Errorf("OpenForm called %d time(s) after a blocking finding", client.openCalls)
			}
		})
	}
}

func TestEPOService_OpenForm_NonBlockingTypesPass(t *testing.T) {
	client := &fakeEPOClient{
		findings: []epo.Finding{{Type: epo.TypePassable}, {Type: epo.TypeNonCritical}, {Type: epo.TypeInfo}},
		link:     "https://mojedane.gov.cz/dpr/x",
	}
	svc := newEPOServiceForTest(&fakeEPOSource{data: []byte("<Pisemnost/>")}, client)

	got, err := svc.OpenForm(context.Background(), EPOKindVATReturn, 1)
	if err != nil {
		t.Fatalf("OpenForm: %v", err)
	}
	if len(got.Warnings) != 2 {
		t.Fatalf("warnings = %d, want 2 (info findings are dropped)", len(got.Warnings))
	}
	for _, w := range got.Warnings {
		if w.Type != epo.TypePassable && w.Type != epo.TypeNonCritical {
			t.Errorf("warning of type %q, want only P and N", w.Type)
		}
	}
}

func TestEPOService_OpenForm_UpstreamErrors(t *testing.T) {
	t.Run("check fails", func(t *testing.T) {
		client := &fakeEPOClient{checkErr: epo.ErrEPOTimeout}
		svc := newEPOServiceForTest(&fakeEPOSource{data: []byte("<Pisemnost/>")}, client)

		if _, err := svc.OpenForm(context.Background(), EPOKindVATReturn, 1); !errors.Is(err, epo.ErrEPOTimeout) {
			t.Fatalf("err = %v, want ErrEPOTimeout", err)
		}
		if client.openCalls != 0 {
			t.Errorf("OpenForm must not run when the check failed")
		}
	})

	t.Run("open fails", func(t *testing.T) {
		client := &fakeEPOClient{findings: nil, openErr: epo.ErrEPOUnavailable}
		svc := newEPOServiceForTest(&fakeEPOSource{data: []byte("<Pisemnost/>")}, client)

		if _, err := svc.OpenForm(context.Background(), EPOKindVATReturn, 1); !errors.Is(err, epo.ErrEPOUnavailable) {
			t.Fatalf("err = %v, want ErrEPOUnavailable", err)
		}
	})

	t.Run("open rejected", func(t *testing.T) {
		rejected := &epo.RejectedError{Findings: []epo.Finding{{Type: epo.TypeCritical}}}
		client := &fakeEPOClient{openErr: rejected}
		svc := newEPOServiceForTest(&fakeEPOSource{data: []byte("<Pisemnost/>")}, client)

		_, err := svc.OpenForm(context.Background(), EPOKindVATReturn, 1)
		var got *epo.RejectedError
		if !errors.As(err, &got) || len(got.Findings) != 1 {
			t.Fatalf("err = %v, want *epo.RejectedError with one finding", err)
		}
	})
}

func TestEPOService_OpenForm_InputErrors(t *testing.T) {
	client := &fakeEPOClient{}
	src := &fakeEPOSource{data: []byte("<Pisemnost/>")}
	svc := newEPOServiceForTest(src, client)

	if _, err := svc.OpenForm(context.Background(), EPOKind("social-insurance"), 1); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("unknown kind err = %v, want ErrInvalidInput", err)
	}
	if _, err := svc.OpenForm(context.Background(), EPOKindVATReturn, 0); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("zero id err = %v, want ErrInvalidInput", err)
	}

	src.err = domain.ErrNotFound
	if _, err := svc.OpenForm(context.Background(), EPOKindVATReturn, 5); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("source error err = %v, want ErrNotFound", err)
	}
	if client.checkCalls != 0 {
		t.Errorf("EPO called %d time(s) on input errors", client.checkCalls)
	}
}
