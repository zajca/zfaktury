package desktop

import "testing"

func TestDownloadName(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		fallback string
		want     string
	}{
		{"header name wins", "attachment; filename=dph-2025-03.xml", "x.xml", "dph-2025-03.xml"},
		{"quoted name", `attachment; filename="kh-2025-q1.xml"`, "x.xml", "kh-2025-q1.xml"},
		{"extended name", "attachment; filename*=UTF-8''dpfo-2025.xml", "x.xml", "dpfo-2025.xml"},
		{"missing header", "", "x.xml", "x.xml"},
		{"no filename parameter", "attachment", "x.xml", "x.xml"},
		{"malformed header", "attachment; filename=", "x.xml", "x.xml"},
		{"dot-dot rejected", `attachment; filename=".."`, "x.xml", "x.xml"},
		{"trailing dot-dot rejected", `attachment; filename="a/.."`, "x.xml", "x.xml"},
		{"path stripped", `attachment; filename="../../etc/passwd"`, "x.xml", "passwd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := downloadName(tt.header, tt.fallback); got != tt.want {
				t.Errorf("downloadName(%q, %q) = %q, want %q", tt.header, tt.fallback, got, tt.want)
			}
		})
	}
}
