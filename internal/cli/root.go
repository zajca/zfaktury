package cli

import (
	"github.com/spf13/cobra"
	"github.com/zajca/zfaktury/internal/version"
)

var configFile string

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

func init() {
	rootCmd.Version = version.Full()
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to config file (default: ~/.zfaktury/config.toml)")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
