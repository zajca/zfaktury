package domain

import "time"

// Backup trigger types.
const (
	BackupTriggerManual    = "manual"
	BackupTriggerScheduled = "scheduled"
	BackupTriggerCLI       = "cli"
)

// Backup status types.
const (
	BackupStatusRunning   = "running"
	BackupStatusCompleted = "completed"
	BackupStatusFailed    = "failed"
)

// BackupRecord represents a backup operation in the history.
type BackupRecord struct {
	ID                 int64
	Filename           string
	Status             string
	Trigger            string
	Destination        string
	SizeBytes          int64
	FileCount          int
	DBMigrationVersion int64
	DurationMs         int64
	ErrorMessage       string
	CreatedAt          time.Time
	CompletedAt        *time.Time
}
