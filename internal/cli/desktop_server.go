//go:build server

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.Flags().IntVar(&servePort, "port", 8080, "HTTP server port")
}

func runDesktop(cmd *cobra.Command, args []string) error {
	fmt.Println("Desktop mode not available in server build. Starting HTTP server instead.")
	serveInitConfig = true
	return serveCmd.RunE(cmd, args)
}
