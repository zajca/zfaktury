package cli

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zajca/zfaktury/internal/config"
)

var (
	restoreTarget string
	restoreForce  bool
	restoreDryRun bool
)

func init() {
	restoreCmd.Flags().StringVar(&restoreTarget, "target", "", "Target directory for restore (default: config DataDir)")
	restoreCmd.Flags().BoolVar(&restoreForce, "force", false, "Overwrite existing data without confirmation")
	restoreCmd.Flags().BoolVar(&restoreDryRun, "dry-run", false, "Show what would be restored without actually doing it")
	rootCmd.AddCommand(restoreCmd)
}

var restoreCmd = &cobra.Command{
	Use:   "restore <file>",
	Short: "Restore data from a backup archive",
	Long:  "Restore the database and documents from a previously created backup .tar.gz archive.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRestore(args[0])
	},
}

// restoreMeta represents the metadata stored in backup-meta.json inside the archive.
type restoreMeta struct {
	AppVersion         string `json:"app_version"`
	DBMigrationVersion int64  `json:"db_migration_version"`
	CreatedAt          string `json:"created_at"`
	FileCount          int    `json:"file_count"`
	DBSizeBytes        int64  `json:"db_size_bytes"`
}

func runRestore(archivePath string) error {
	// Determine target directory.
	targetDir := restoreTarget
	if targetDir == "" {
		cfgPath, err := config.Resolve(configFile, false)
		if err != nil {
			return err
		}
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		targetDir = cfg.DataDir
	}
	targetDir = config.ExpandHome(targetDir)

	// Check that the server is not running by looking for a lock file.
	lockFile := filepath.Join(targetDir, ".zfaktury.lock")
	if _, err := os.Stat(lockFile); err == nil {
		return fmt.Errorf("server appears to be running (lock file exists: %s). Stop the server before restoring", lockFile)
	}

	return restoreFromArchive(archivePath, targetDir, restoreForce, restoreDryRun)
}

func restoreFromArchive(archivePath, targetDir string, force, dryRun bool) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("opening gzip reader: %w", err)
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)

	// First pass: read metadata and collect file list.
	var meta *restoreMeta
	type fileEntry struct {
		name string
		size int64
	}
	var files []fileEntry

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("reading archive: %w", err)
		}

		// Sanitize path to prevent directory traversal.
		cleanName := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanName, "..") || strings.HasPrefix(cleanName, "/") {
			return fmt.Errorf("archive contains suspicious path: %s", header.Name)
		}

		if cleanName == "backup-meta.json" {
			data, err := io.ReadAll(io.LimitReader(tr, 1<<20)) // 1 MB limit for meta
			if err != nil {
				return fmt.Errorf("reading backup metadata: %w", err)
			}
			meta = &restoreMeta{}
			if err := json.Unmarshal(data, meta); err != nil {
				return fmt.Errorf("parsing backup metadata: %w", err)
			}
			continue
		}

		if header.Typeflag == tar.TypeReg {
			files = append(files, fileEntry{name: cleanName, size: header.Size})
		}
	}

	if meta == nil {
		return fmt.Errorf("archive does not contain backup-meta.json -- not a valid ZFaktury backup")
	}

	// Print backup info.
	fmt.Printf("Restoring from: %s\n", filepath.Base(archivePath))
	fmt.Printf("  App version:       %s\n", meta.AppVersion)
	fmt.Printf("  Migration version: %d\n", meta.DBMigrationVersion)
	fmt.Printf("  Created at:        %s\n", meta.CreatedAt)
	fmt.Printf("  Files:             %d\n", meta.FileCount)
	fmt.Printf("  DB size:           %s\n", formatBytes(meta.DBSizeBytes))
	fmt.Println()

	if dryRun {
		fmt.Println("Dry run -- no files were modified.")
		fmt.Println("\nFiles in archive:")
		for _, fe := range files {
			fmt.Printf("  %s (%s)\n", fe.name, formatBytes(fe.size))
		}
		return nil
	}

	// Check if target directory has existing data.
	dbPath := filepath.Join(targetDir, "zfaktury.db")
	if _, err := os.Stat(dbPath); err == nil && !force {
		return fmt.Errorf("existing database found at %s. Use --force to overwrite", dbPath)
	}

	// Re-open the archive for extraction.
	_ = f.Close()
	f, err = os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("re-opening archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	gzr, err = gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("re-opening gzip reader: %w", err)
	}
	defer func() { _ = gzr.Close() }()

	tr = tar.NewReader(gzr)

	var extractedCount int

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("reading archive: %w", err)
		}

		cleanName := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanName, "..") || strings.HasPrefix(cleanName, "/") {
			continue // skip suspicious paths
		}

		// Skip metadata file during extraction.
		if cleanName == "backup-meta.json" {
			continue
		}

		// Map database.db to zfaktury.db in target dir.
		var destPath string
		if cleanName == "database.db" {
			destPath = filepath.Join(targetDir, "zfaktury.db")
		} else {
			destPath = filepath.Join(targetDir, cleanName)
		}

		// Canonical tar slip protection: ensure destPath is inside targetDir.
		absTarget, _ := filepath.Abs(targetDir)
		absDest, _ := filepath.Abs(destPath)
		if !strings.HasPrefix(absDest, absTarget+string(filepath.Separator)) && absDest != absTarget {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(destPath, 0o755); err != nil {
				return fmt.Errorf("creating directory %s: %w", destPath, err)
			}
		case tar.TypeReg:
			if err := extractFile(tr, destPath, header.Mode); err != nil {
				return fmt.Errorf("extracting %s: %w", cleanName, err)
			}
			extractedCount++
		}
	}

	fmt.Printf("Restored %d files to %s\n", extractedCount, targetDir)

	return nil
}

// extractFile writes a single file from the tar reader to the destination path.
func extractFile(r io.Reader, destPath string, mode int64) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	fileMode := os.FileMode(mode)
	if fileMode == 0 {
		fileMode = 0o644
	}

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer func() { _ = out.Close() }()

	// Limit extraction to 10 GB per file to prevent decompression bombs.
	const maxFileSize = 10 << 30
	if _, err := io.Copy(out, io.LimitReader(r, maxFileSize)); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}
