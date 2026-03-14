package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/zajca/zfaktury/internal/config"
	"github.com/zajca/zfaktury/internal/database"
	"github.com/zajca/zfaktury/internal/domain"
	"github.com/zajca/zfaktury/internal/repository"
	"github.com/zajca/zfaktury/internal/service"
)

var backupOutput string

func init() {
	backupCmd.Flags().StringVarP(&backupOutput, "output", "o", "", "Override backup destination directory")
	rootCmd.AddCommand(backupCmd)
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a backup of the database and documents",
	Long:  "Create a compressed backup archive containing the SQLite database and all uploaded documents.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBackup()
	},
}

func runBackup() error {
	cfgPath, err := config.Resolve(configFile, false)
	if err != nil {
		return err
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	db, err := database.New(cfg)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	// Override backup config with CLI flags.
	backupCfg := cfg.Backup
	if backupOutput != "" {
		backupCfg.Destination = backupOutput
	}

	backupHistoryRepo := repository.NewBackupHistoryRepository(db)

	// CLI backup always uses local storage (--output flag determines local path).
	destDir := backupCfg.Destination
	if destDir == "" {
		destDir = cfg.BackupDestination()
	}
	backupStorage := service.NewLocalStorage(destDir)
	backupSvc := service.NewBackupService(backupHistoryRepo, db, backupCfg, cfg.DataDir, backupStorage)

	start := time.Now()
	fmt.Println("Creating backup...")

	record, err := backupSvc.CreateBackup(context.Background(), domain.BackupTriggerCLI)
	if err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	duration := time.Since(start)

	fmt.Printf("\nBackup completed successfully!\n")
	fmt.Printf("  File:     %s\n", record.Filename)
	fmt.Printf("  Size:     %s\n", formatBytes(record.SizeBytes))
	fmt.Printf("  Files:    %d\n", record.FileCount)
	fmt.Printf("  Duration: %s\n", duration.Round(time.Millisecond))

	return nil
}

// formatBytes formats a byte count into a human-readable string.
func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)

	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
