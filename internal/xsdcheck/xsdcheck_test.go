package xsdcheck

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const goldenPath = "../annualtaxxml/testdata/social_insurance_regular.golden.xml"

// requireXmllint skips when xmllint is missing, except under CI, where a
// missing xmllint is a failure so the check can never silently stop running.
func requireXmllint(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("xmllint"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatalf("xmllint is required in CI: %v", err)
		}
		t.Skip("xmllint not available; install libxml2 to run the XSD check tests")
	}
}

func newTestChecker(t *testing.T, timeout time.Duration, maxOutput int) (*Checker, string) {
	t.Helper()
	workDir := filepath.Join(t.TempDir(), "xsdcheck-work")
	c, err := New(Config{WorkDir: workDir, Timeout: timeout, MaxOutputBytes: maxOutput})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, workDir
}

func readGolden(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden XML: %v", err)
	}
	return data
}

// fakeXmllint puts an executable shell script named xmllint first on PATH.
func fakeXmllint(t *testing.T, script string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script stand-in needs a POSIX shell")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "xmllint"), []byte("#!/bin/sh\n"+script+"\n"), 0o700); err != nil {
		t.Fatalf("writing fake xmllint: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+"/usr/bin"+string(os.PathListSeparator)+"/bin")
}

func assertWorkDirEmpty(t *testing.T, workDir string) {
	t.Helper()
	entries, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatalf("reading work dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("work dir not cleaned up, %d entries left", len(entries))
	}
}

func TestNew_ConfigValidation(t *testing.T) {
	good := Config{WorkDir: t.TempDir(), Timeout: time.Second, MaxOutputBytes: 10, WaitDelay: time.Second}
	if _, err := New(good); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	cases := map[string]func(*Config){
		"work dir":   func(c *Config) { c.WorkDir = "" },
		"timeout":    func(c *Config) { c.Timeout = -1 },
		"max output": func(c *Config) { c.MaxOutputBytes = -1 },
		"wait delay": func(c *Config) { c.WaitDelay = -1 },
	}
	for name, mutate := range cases {
		cfg := good
		mutate(&cfg)
		if _, err := New(cfg); !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("%s: err = %v, want ErrInvalidConfig", name, err)
		}
	}
}

func TestNew_ZeroValuesKeepDefaults(t *testing.T) {
	c, err := New(Config{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.cfg.Timeout != DefaultTimeout || c.cfg.MaxOutputBytes != DefaultMaxOutputBytes || c.cfg.WaitDelay != DefaultWaitDelay {
		t.Errorf("defaults not applied: %+v", c.cfg)
	}

	c, err = New(Config{WorkDir: t.TempDir(), Timeout: 3 * time.Second, MaxOutputBytes: 77, WaitDelay: 5 * time.Second})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if c.cfg.Timeout != 3*time.Second || c.cfg.MaxOutputBytes != 77 || c.cfg.WaitDelay != 5*time.Second {
		t.Errorf("explicit values overridden: %+v", c.cfg)
	}
}

func TestCheck_WorksWithDefaults(t *testing.T) {
	requireXmllint(t)
	c, err := New(Config{WorkDir: filepath.Join(t.TempDir(), "work")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := c.Check(context.Background(), SchemaOSVC25, readGolden(t)); err != nil {
		t.Fatalf("Check with default limits: %v", err)
	}
}

// Exit code 4 with a "failed to load" or "Can't open" message is an unreadable
// input, not an invalid document.
func TestCheck_UnreadableInputIsExecError(t *testing.T) {
	for _, msg := range []string{"I/O warning : failed to load external entity", "Can't open file"} {
		fakeXmllint(t, `echo "`+msg+`" 1>&2; exit 4`)
		c, _ := newTestChecker(t, 30*time.Second, 500)
		err := c.Check(context.Background(), SchemaOSVC25, []byte("<a/>"))
		if !errors.Is(err, ErrExec) || errors.Is(err, ErrInvalid) {
			t.Errorf("%q: err = %v, want ErrExec and not ErrInvalid", msg, err)
		}
	}
}

func TestCheck_NoLocalPathsInMessages(t *testing.T) {
	// The stand-in echoes the paths it was given, as xmllint does.
	fakeXmllint(t, `echo "$5:1: parser error" 1>&2; echo "$4 $5" 1>&2; exit 3`)
	c, workDir := newTestChecker(t, 30*time.Second, 2000)
	err := c.Check(context.Background(), SchemaOSVC25, []byte("<a/>"))
	var invalid *InvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("err = %v, want *InvalidError", err)
	}
	if strings.Contains(invalid.Messages, workDir) || strings.Contains(invalid.Messages, string(os.PathSeparator)) {
		t.Errorf("messages leak a local path: %q", invalid.Messages)
	}
	if !strings.Contains(invalid.Messages, "document.xml:1: parser error") {
		t.Errorf("messages lost the file name: %q", invalid.Messages)
	}

	fakeXmllint(t, `echo "$5 broke" 1>&2; exit 5`)
	err = c.Check(context.Background(), SchemaOSVC25, []byte("<a/>"))
	if !errors.Is(err, ErrExec) {
		t.Fatalf("err = %v, want ErrExec", err)
	}
	if strings.Contains(err.Error(), workDir) {
		t.Errorf("ErrExec text leaks a local path: %v", err)
	}
}

func TestCheck_WorkDirErrorHasNoPath(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	fakeXmllint(t, "exit 0")
	c, err := New(Config{WorkDir: filepath.Join(blocker, "sub")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err = c.Check(context.Background(), SchemaOSVC25, []byte("<a/>"))
	if !errors.Is(err, ErrExec) {
		t.Fatalf("err = %v, want ErrExec", err)
	}
	if strings.Contains(err.Error(), blocker) {
		t.Errorf("error leaks a local path: %v", err)
	}
}

func TestCheck_ValidDocument(t *testing.T) {
	requireXmllint(t)
	c, workDir := newTestChecker(t, 30*time.Second, 4096)
	if err := c.Check(context.Background(), SchemaOSVC25, readGolden(t)); err != nil {
		t.Fatalf("Check on the golden OSVC25 XML: %v", err)
	}
	assertWorkDirEmpty(t, workDir)
}

func TestCheck_InvalidDocument(t *testing.T) {
	requireXmllint(t)
	c, workDir := newTestChecker(t, 30*time.Second, 4096)
	// "dep" (the OSSZ code) is a required attribute of prehledosvc.
	broken := strings.Replace(string(readGolden(t)), ` dep="775"`, "", 1)
	if broken == string(readGolden(t)) {
		t.Fatal("test setup: replacement did not change the XML")
	}

	err := c.Check(context.Background(), SchemaOSVC25, []byte(broken))
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	var invalid *InvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("err %T is not *InvalidError", err)
	}
	if !strings.Contains(invalid.Messages, "'dep'") {
		t.Errorf("messages do not mention the bad attribute: %q", invalid.Messages)
	}
	assertWorkDirEmpty(t, workDir)
}

func TestCheck_MalformedXML(t *testing.T) {
	requireXmllint(t)
	c, workDir := newTestChecker(t, 30*time.Second, 4096)
	err := c.Check(context.Background(), SchemaOSVC25, []byte("<OSVC>"))
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v, want ErrInvalid", err)
	}
	assertWorkDirEmpty(t, workDir)
}

func TestCheck_EmptyXML(t *testing.T) {
	c, _ := newTestChecker(t, time.Second, 10)
	if err := c.Check(context.Background(), SchemaOSVC25, nil); !errors.Is(err, ErrExec) {
		t.Errorf("err = %v, want ErrExec", err)
	}
}

func TestCheck_UnknownSchema(t *testing.T) {
	fakeXmllint(t, "exit 0")
	c, workDir := newTestChecker(t, time.Second, 10)
	err := c.Check(context.Background(), Schema{name: "nope", file: "nope.xsd"}, []byte("<a/>"))
	if !errors.Is(err, ErrExec) {
		t.Errorf("err = %v, want ErrExec", err)
	}
	assertWorkDirEmpty(t, workDir)
}

func TestCheck_XmllintMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	c, workDir := newTestChecker(t, time.Second, 10)
	err := c.Check(context.Background(), SchemaOSVC25, []byte("<a/>"))
	if !errors.Is(err, ErrXmllintMissing) {
		t.Fatalf("err = %v, want ErrXmllintMissing", err)
	}
	if _, statErr := os.Stat(workDir); statErr == nil {
		assertWorkDirEmpty(t, workDir)
	}
}

func TestCheck_Timeout(t *testing.T) {
	fakeXmllint(t, "exec sleep 30")
	c, workDir := newTestChecker(t, 200*time.Millisecond, 10)
	start := time.Now()
	err := c.Check(context.Background(), SchemaOSVC25, []byte("<a/>"))
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v, want ErrTimeout", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("Check took %v, the child was not killed", elapsed)
	}
	assertWorkDirEmpty(t, workDir)
}

func TestCheck_CallerDeadlineWins(t *testing.T) {
	fakeXmllint(t, "exec sleep 30")
	c, _ := newTestChecker(t, time.Hour, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := c.Check(ctx, SchemaOSVC25, []byte("<a/>"))
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v, want ErrTimeout", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Errorf("Check took %v, the caller deadline was ignored", elapsed)
	}
}

func TestCheck_CallerCancel(t *testing.T) {
	fakeXmllint(t, "exec sleep 30")
	c, _ := newTestChecker(t, time.Hour, 10)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(200*time.Millisecond, cancel)
	err := c.Check(ctx, SchemaOSVC25, []byte("<a/>"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if errors.Is(err, ErrTimeout) || errors.Is(err, ErrInvalid) {
		t.Errorf("cancellation misreported: %v", err)
	}
}

func TestCheck_OutputCap(t *testing.T) {
	fakeXmllint(t, `i=0; while [ $i -lt 2000 ]; do echo "document.xml:1: element x: Schemas validity error : message number $i"; i=$((i+1)); done 1>&2; exit 3`)
	const limit = 500
	c, workDir := newTestChecker(t, 30*time.Second, limit)
	err := c.Check(context.Background(), SchemaOSVC25, []byte("<a/>"))
	var invalid *InvalidError
	if !errors.As(err, &invalid) {
		t.Fatalf("err = %v, want *InvalidError", err)
	}
	if len(invalid.Messages) != limit {
		t.Errorf("len(Messages) = %d, want %d", len(invalid.Messages), limit)
	}
	if !invalid.Truncated {
		t.Error("Truncated = false, want true")
	}
	if !strings.HasPrefix(invalid.Messages, "document.xml:1: element x") {
		t.Errorf("messages do not start with the first xmllint line: %q", invalid.Messages[:40])
	}
	assertWorkDirEmpty(t, workDir)
}

func TestCheck_ShortOutputNotTruncated(t *testing.T) {
	fakeXmllint(t, `echo "boom" 1>&2; exit 3`)
	c, _ := newTestChecker(t, 30*time.Second, 500)
	var invalid *InvalidError
	if err := c.Check(context.Background(), SchemaOSVC25, []byte("<a/>")); !errors.As(err, &invalid) {
		t.Fatalf("err = %v, want *InvalidError", err)
	}
	if invalid.Truncated || strings.TrimSpace(invalid.Messages) != "boom" {
		t.Errorf("got %+v", invalid)
	}
}

func TestCheck_OtherExecFailure(t *testing.T) {
	// Exit code 5 is a schema compilation failure, not an invalid document.
	fakeXmllint(t, `echo "schema broken" 1>&2; exit 5`)
	c, workDir := newTestChecker(t, 30*time.Second, 500)
	err := c.Check(context.Background(), SchemaOSVC25, []byte("<a/>"))
	if !errors.Is(err, ErrExec) {
		t.Fatalf("err = %v, want ErrExec", err)
	}
	if errors.Is(err, ErrInvalid) || errors.Is(err, ErrTimeout) || errors.Is(err, ErrXmllintMissing) {
		t.Errorf("misclassified: %v", err)
	}
	if !strings.Contains(err.Error(), "schema broken") {
		t.Errorf("error lacks xmllint output: %v", err)
	}
	assertWorkDirEmpty(t, workDir)
}

func TestCheck_FixedArguments(t *testing.T) {
	// The stand-in records its arguments, one per line, and reports "valid".
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	fakeXmllint(t, `for a in "$@"; do echo "$a"; done > `+argsFile+`; exit 0`)
	c, _ := newTestChecker(t, 30*time.Second, 500)
	if err := c.Check(context.Background(), SchemaOSVC25, []byte("<a/>")); err != nil {
		t.Fatalf("Check: %v", err)
	}
	raw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("reading recorded args: %v", err)
	}
	args := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(args) != 5 {
		t.Fatalf("args = %q, want 5", args)
	}
	if args[0] != "--noout" || args[1] != "--nonet" || args[2] != "--schema" {
		t.Errorf("flags = %q", args[:3])
	}
	if filepath.Base(args[3]) != "OSVC25.xsd" || filepath.Base(args[4]) != "document.xml" {
		t.Errorf("paths = %q", args[3:])
	}
	if !filepath.IsAbs(args[3]) || !filepath.IsAbs(args[4]) {
		t.Errorf("paths are not absolute: %q", args[3:])
	}
}
