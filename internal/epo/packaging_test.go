package epo

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// unzipSingleEntry opens body as a ZIP archive, requires exactly one entry and
// returns its name and content.
func unzipSingleEntry(t *testing.T, body []byte) (string, []byte) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("request body is not a ZIP archive: %v", err)
	}
	if len(zr.File) != 1 {
		t.Fatalf("archive has %d entries, want exactly 1", len(zr.File))
	}
	rc, err := zr.File[0].Open()
	if err != nil {
		t.Fatalf("opening ZIP entry: %v", err)
	}
	defer func() { _ = rc.Close() }()
	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading ZIP entry: %v", err)
	}
	return zr.File[0].Name, content
}

func TestPackagingZIP_SendsSingleEntryArchive(t *testing.T) {
	calls := map[string]func(c *Client) error{
		"OpenForm": func(c *Client) error {
			_, err := c.OpenForm(context.Background(), []byte(sampleXML), PackagingZIP)
			return err
		},
		"Check": func(c *Client) error {
			_, err := c.Check(context.Background(), []byte(sampleXML), PackagingZIP)
			return err
		},
	}
	bodies := map[string][]byte{
		"OpenForm": fixture(t, "open_form_url.xml"),
		"Check":    fixture(t, "check_warning.xml"),
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			srv, rec := newFake(t, http.StatusOK, bodies[name])
			c := NewClient(WithBaseURL(srv.URL + "/dpr"))

			if err := call(c); err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			entryName, content := unzipSingleEntry(t, rec.body)
			if entryName != ZIPEntryName {
				t.Errorf("entry name = %q, want %q", entryName, ZIPEntryName)
			}
			if !bytes.Equal(content, []byte(sampleXML)) {
				t.Errorf("archived XML differs from the input XML")
			}
			if rec.contentType != "application/octet-stream" {
				t.Errorf("content type = %q, want application/octet-stream", rec.contentType)
			}
			if rec.path != "/dpr/epo_podani" {
				t.Errorf("path = %q", rec.path)
			}
		})
	}
}

func TestPackagingPlain_SendsXMLUnchanged(t *testing.T) {
	srv, rec := newFake(t, http.StatusOK, fixture(t, "open_form_url.xml"))
	c := NewClient(WithBaseURL(srv.URL + "/dpr"))

	if _, err := c.OpenForm(context.Background(), []byte(sampleXML), PackagingPlain); err != nil {
		t.Fatalf("OpenForm: %v", err)
	}
	if !bytes.Equal(rec.body, []byte(sampleXML)) {
		t.Errorf("plain packaging changed the body")
	}
}

func TestPackagingZIP_ValidatesBeforeCompressing(t *testing.T) {
	var calls int
	srv := newCountingServer(t, &calls)
	c := NewClient(WithBaseURL(srv.URL), WithMaxRequestBytes(64))

	// Highly compressible but over the limit uncompressed: the limit applies
	// to the XML, not to the archive.
	oversized := append([]byte("<Pisemnost>"), bytes.Repeat([]byte("a"), 1000)...)
	if _, err := c.Check(context.Background(), oversized, PackagingZIP); !errors.Is(err, ErrInvalidXML) {
		t.Errorf("oversized XML err = %v, want ErrInvalidXML", err)
	}
	if _, err := c.OpenForm(context.Background(), []byte("not xml"), PackagingZIP); !errors.Is(err, ErrInvalidXML) {
		t.Errorf("invalid XML err = %v, want ErrInvalidXML", err)
	}
	if calls != 0 {
		t.Errorf("fake received %d request(s), want none", calls)
	}
}

func TestPackagingZIP_ArchiveOverLimitIsRefused(t *testing.T) {
	var calls int
	srv := newCountingServer(t, &calls)
	// The XML fits the limit but the archive overhead (headers, directory)
	// pushes the archive past it.
	xmlData := []byte(`<Pisemnost/>`)
	c := NewClient(WithBaseURL(srv.URL), WithMaxRequestBytes(int64(len(xmlData))))

	if _, err := c.Check(context.Background(), xmlData, PackagingZIP); !errors.Is(err, ErrInvalidXML) {
		t.Errorf("err = %v, want ErrInvalidXML", err)
	}
	if calls != 0 {
		t.Errorf("fake received %d request(s), want none", calls)
	}
}

func TestPackaging_UnknownValueIsRefused(t *testing.T) {
	var calls int
	srv := newCountingServer(t, &calls)
	c := NewClient(WithBaseURL(srv.URL))

	if _, err := c.Check(context.Background(), []byte(sampleXML), Packaging(99)); !errors.Is(err, ErrInvalidXML) {
		t.Errorf("err = %v, want ErrInvalidXML", err)
	}
	if calls != 0 {
		t.Errorf("fake received %d request(s), want none", calls)
	}
}

// newCountingServer starts a server that only counts requests, so refused
// input is provably never sent.
func newCountingServer(t *testing.T, calls *int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		*calls++
	}))
	t.Cleanup(srv.Close)
	return srv
}
