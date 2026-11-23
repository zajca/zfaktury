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
		want int
	}{
		{name: "flag not given", args: nil, want: 0},
		{name: "flag given", args: []string{"--port", "7000"}, want: 7000},
		{name: "flag given with the old default value", args: []string{"--port", "8080"}, want: 8080},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, port := newPortTestCommand()
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("ParseFlags() error: %v", err)
			}
			if got := portOverride(cmd, *port); got != tt.want {
				t.Errorf("portOverride() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestServeCommandPortFlagHasNoDefault(t *testing.T) {
	flag := serveCmd.Flags().Lookup("port")
	if flag == nil {
		t.Fatal("serve command has no --port flag")
	}
	if flag.DefValue != "0" {
		t.Errorf("--port default = %q, want 0 so that server.port applies", flag.DefValue)
	}
}
