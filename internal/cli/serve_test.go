package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func newPortTestCommand() (*cobra.Command, *int) {
	var port int
	cmd := &cobra.Command{Use: "serve"}
	cmd.Flags().IntVar(&port, "port", 0, "HTTP server port")
	return cmd, &port
}

func TestPortOverride(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want *int
	}{
		{name: "flag not given", args: nil, want: nil},
		{name: "flag given", args: []string{"--port", "7000"}, want: intPtr(7000)},
		{name: "flag given with the old default value", args: []string{"--port", "8080"}, want: intPtr(8080)},
		{name: "flag given as zero", args: []string{"--port", "0"}, want: intPtr(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, port := newPortTestCommand()
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("ParseFlags() error: %v", err)
			}
			got := portOverride(cmd, *port)
			switch {
			case tt.want == nil && got != nil:
				t.Errorf("portOverride() = %d, want nil", *got)
			case tt.want != nil && got == nil:
				t.Errorf("portOverride() = nil, want %d", *tt.want)
			case tt.want != nil && *got != *tt.want:
				t.Errorf("portOverride() = %d, want %d", *got, *tt.want)
			}
		})
	}
}

func intPtr(v int) *int { return &v }
