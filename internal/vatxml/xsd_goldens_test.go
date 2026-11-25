package vatxml

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zajca/zfaktury/internal/xsdcheck"
)

// TestGoldensValidateAgainstEPOSchema feeds every committed DP3, KH and VIES
// golden through xmllint against the EPO XSD it targets. It fails when a
// generator drifts out of the schema, so the goldens cannot silently rot.
//
// The test skips when xmllint is missing outside CI, the same way the
// internal/xsdcheck tests do.
func TestGoldensValidateAgainstEPOSchema(t *testing.T) {
	if _, err := exec.LookPath("xmllint"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("xmllint is required in CI: %v", err)
		}
		t.Skip("xmllint not available; install libxml2 to run the XSD check")
	}

	checker, err := xsdcheck.New(xsdcheck.Config{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatalf("creating the XSD checker: %v", err)
	}

	paths, err := filepath.Glob(filepath.Join("testdata", "*.golden.xml"))
	if err != nil {
		t.Fatalf("globbing the golden files: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no golden files found in testdata")
	}

	for _, path := range paths {
		schema, ok := epoSchemaForGolden(filepath.Base(path))
		if !ok {
			t.Errorf("%s: no EPO schema mapped for this golden", path)
			continue
		}
		xmlData, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		if err := checker.Check(context.Background(), schema, xmlData); err != nil {
			t.Errorf("%s does not validate against %s: %v", path, schema, err)
		}
	}
}

// epoSchemaForGolden maps a golden file name to the EPO schema it must satisfy.
func epoSchemaForGolden(name string) (xsdcheck.Schema, bool) {
	switch {
	case strings.HasPrefix(name, "vat_return_"):
		return xsdcheck.SchemaDPHDP3, true
	case strings.HasPrefix(name, "control_statement_"):
		return xsdcheck.SchemaDPHKH1, true
	case strings.HasPrefix(name, "vies_summary_"):
		return xsdcheck.SchemaDPHSHV, true
	default:
		return xsdcheck.Schema{}, false
	}
}
