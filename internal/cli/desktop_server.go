//go:build server

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.Flags().IntVar(&servePort, "port", 0, "HTTP server port (overrides server.port from the config; 0 = random free port)")
}

func runDesktop(cmd *cobra.Command, args []string) error {
	fmt.Println("Desktop mode not available in server build. Starting HTTP server instead.")
	serveInitConfig = true
	return serveCmd.RunE(cmd, args)
}
