package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "zfaktury",
	Short: "Invoice and tax management for Czech sole proprietors",
	Long: `ZFaktury is a self-hosted invoice and tax management application
designed specifically for Czech sole proprietors (OSVČ).

It provides invoice creation, expense tracking, tax calculations,
and integration with Czech financial services (ARES, FIO Bank, CNB).`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
