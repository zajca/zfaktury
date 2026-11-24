// Package xsdcheck validates generated XML against an embedded XSD by running
// the system xmllint. It starts no shell, passes fixed arguments only and
// never touches the network.
package xsdcheck

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	csszschemas "github.com/zajca/zfaktury/docs/xml-schemas/cssz"
)

var (
	// ErrXmllintMissing is returned when xmllint is not on PATH. Callers tell
	// the user to install libxml2 or to file manually.
	ErrXmllintMissing = errors.New("xmllint is not installed")
	// ErrTimeout is returned when xmllint did not finish in time.
	ErrTimeout = errors.New("xmllint timed out")
	// ErrInvalid is matched by *InvalidError: the XML does not conform.
	ErrInvalid = errors.New("XML does not conform to the schema")
	// ErrExec is returned when xmllint could not be run or failed for a
	// reason other than the XML being invalid.
	ErrExec = errors.New("running xmllint failed")
	// ErrInvalidConfig is returned by New for a missing or non-positive
	// setting.
	ErrInvalidConfig = errors.New("invalid xsdcheck configuration")
)

// Schema is an embedded XSD entry point.
type Schema struct {
	name string
	file string
}

// SchemaOSVC25 is the ČSSZ OSVČ overview schema (namespace
// http://schemas.cssz.cz/OSVC2025).
var SchemaOSVC25 = Schema{name: "OSVC25", file: "OSVC25.xsd"}

// String returns the schema name.
func (s Schema) String() string { return s.name }

// InvalidError carries the xmllint messages for a non-conforming document.
type InvalidError struct {
	// Messages is the xmllint output, cut at the configured size cap.
	Messages string
	// Truncated is true when the output exceeded the cap.
	Truncated bool
}

func (e *InvalidError) Error() string { return ErrInvalid.Error() + ": " + e.Messages }

// Is makes errors.Is(err, ErrInvalid) true.
func (e *InvalidError) Is(target error) bool { return target == ErrInvalid }

// Config holds the required settings of a Checker. There are no defaults:
// every field must be set.
type Config struct {
	// WorkDir is the parent of the per-call temporary directories, for
	// example a directory under the data dir.
	WorkDir string
	// Timeout is the longest a single xmllint run may take. A shorter
	// deadline on the caller's context wins.
	Timeout time.Duration
	// MaxOutputBytes caps the captured xmllint output.
	MaxOutputBytes int
}

// Checker runs xmllint against embedded schemas.
type Checker struct {
	cfg Config
}

// New validates cfg and returns a Checker.
func New(cfg Config) (*Checker, error) {
	if strings.TrimSpace(cfg.WorkDir) == "" {
		return nil, fmt.Errorf("%w: work dir is required", ErrInvalidConfig)
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("%w: timeout must be positive", ErrInvalidConfig)
	}
	if cfg.MaxOutputBytes <= 0 {
		return nil, fmt.Errorf("%w: max output bytes must be positive", ErrInvalidConfig)
	}
	return &Checker{cfg: cfg}, nil
}

// xmllint exit codes that mean "the document is bad" (libxml2 2.9 and later):
// 3 is a schema validation error, 4 is a document that cannot be parsed. Every
// other code (for example 5, schema compilation) is a failure of the check.
const (
	exitValidation = 3
	exitUnreadable = 4
)

// Check validates xml against schema. It returns nil for a conforming
// document, *InvalidError (matching ErrInvalid) for a non-conforming one,
// ErrXmllintMissing, ErrTimeout, a wrapped context error when the caller
// cancelled, or an error wrapping ErrExec.
func (c *Checker) Check(ctx context.Context, schema Schema, xml []byte) error {
	if len(xml) == 0 {
		return fmt.Errorf("%w: empty XML", ErrExec)
	}
	bin, err := exec.LookPath("xmllint")
	if err != nil {
		return ErrXmllintMissing
	}

	workDir, err := filepath.Abs(c.cfg.WorkDir)
	if err != nil {
		return fmt.Errorf("%w: resolving work dir: %v", ErrExec, err)
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return fmt.Errorf("%w: creating work dir: %v", ErrExec, err)
	}
	dir, err := os.MkdirTemp(workDir, "xsdcheck-")
	if err != nil {
		return fmt.Errorf("%w: creating temp dir: %v", ErrExec, err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	xsdPath, err := writeSchemas(dir, schema)
	if err != nil {
		return err
	}
	xmlPath := filepath.Join(dir, "document.xml")
	if err := os.WriteFile(xmlPath, xml, 0o600); err != nil {
		return fmt.Errorf("%w: writing XML: %v", ErrExec, err)
	}

	runCtx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	out := &cappedBuffer{limit: c.cfg.MaxOutputBytes}
	// Fixed arguments: --nonet forbids any network fetch, and both paths are
	// absolute and generated here.
	cmd := exec.CommandContext(runCtx, bin, "--noout", "--nonet", "--schema", xsdPath, xmlPath)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Dir = dir
	// Bounds the wait for output pipes held open by an orphaned child.
	cmd.WaitDelay = time.Second

	runErr := cmd.Run()
	if runErr == nil {
		return nil
	}
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		if ctx.Err() == nil {
			return ErrTimeout
		}
		return fmt.Errorf("%w: %w", ErrTimeout, ctx.Err())
	}
	if ctx.Err() != nil {
		return fmt.Errorf("xmllint cancelled: %w", ctx.Err())
	}
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		if code := exitErr.ExitCode(); code == exitValidation || code == exitUnreadable {
			return &InvalidError{Messages: out.String(), Truncated: out.truncated}
		}
		return fmt.Errorf("%w: exit code %d: %s", ErrExec, exitErr.ExitCode(), out.String())
	}
	return fmt.Errorf("%w: %v", ErrExec, runErr)
}

// writeSchemas copies the embedded XSD files into dir, so imports resolve
// next to the entry point, and returns the entry point path.
func writeSchemas(dir string, schema Schema) (string, error) {
	entries, err := fs.ReadDir(csszschemas.Files, ".")
	if err != nil {
		return "", fmt.Errorf("%w: reading embedded schemas: %v", ErrExec, err)
	}
	found := false
	for _, e := range entries {
		data, err := csszschemas.Files.ReadFile(e.Name())
		if err != nil {
			return "", fmt.Errorf("%w: reading embedded %s: %v", ErrExec, e.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0o600); err != nil {
			return "", fmt.Errorf("%w: writing schema %s: %v", ErrExec, e.Name(), err)
		}
		if e.Name() == schema.file {
			found = true
		}
	}
	if !found {
		return "", fmt.Errorf("%w: schema %q is not embedded", ErrExec, schema.name)
	}
	return filepath.Join(dir, schema.file), nil
}

// cappedBuffer keeps the first limit bytes written to it and discards the
// rest, so a chatty child never blocks on a full pipe or exhausts memory.
type cappedBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	room := b.limit - b.buf.Len()
	if room < len(p) {
		b.truncated = true
	}
	if room > 0 {
		b.buf.Write(p[:min(room, len(p))])
	}
	return len(p), nil
}

func (b *cappedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
