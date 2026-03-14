# RFC-011: Backup & Sync

**Status:** Draft
**Date:** 2026-03-13

## Summary

Safe backup of ZFaktury data (SQLite database + document files) to `.tar.gz` archives with metadata, scheduled retention, and restore capability. Sync is handled externally via `ZFAKTURY_DATA_DIR` pointed at a synced folder, with instance locking to prevent concurrent writes.

## Background

ZFaktury stores all data in a single SQLite database and document files on disk. No backup functionality exists today. If the disk fails, all invoices, expenses, contacts, tax filings, and uploaded documents are permanently lost.

Czech OSVC are legally required to retain invoices and tax documents for 10 years (3 years for income tax, 10 years for VAT). A reliable backup mechanism is essential.

### Requirements

1. Create consistent snapshots of the database (no partial writes)
2. Include all document files (uploaded invoices, receipts, generated PDFs)
3. Support scheduled automatic backups with retention policy
4. Provide restore from backup archive
5. Support external sync tools (Syncthing, NAS, Nextcloud) without built-in sync complexity

## Design

### Backup Strategy

#### SQLite Snapshot

Use `VACUUM INTO` for creating a safe, consistent copy of the database. This works with `modernc.org/sqlite` (no CGO required) and produces a standalone database file that is:

- Consistent (point-in-time snapshot, no WAL or journal files needed)
- Compact (defragmented, no free pages)
- Safe to copy while the source database is in use

```go
func (r *BackupRepo) CreateSnapshot(ctx context.Context, destPath string) error {
    _, err := r.db.ExecContext(ctx, "VACUUM INTO ?", destPath)
    return err
}
```

#### Archive Format

The backup is a `.tar.gz` archive containing:

```
zfaktury-backup-2026-03-13T14-30-00.tar.gz
├── backup-meta.json
├── zfaktury.db
└── documents/
    ├── invoices/
    │   └── ...
    └── expenses/
        └── ...
```

#### Metadata File

`backup-meta.json` is included at the root of the archive:

```json
{
  "version": "0.12.0",
  "migration_version": 24,
  "created_at": "2026-03-13T14:30:00Z",
  "db_checksum_sha256": "abc123...",
  "db_size_bytes": 1048576,
  "documents_count": 42,
  "documents_size_bytes": 52428800,
  "archive_size_bytes": 31457280
}
```

Fields:
- `version` — application version at backup time
- `migration_version` — current goose migration version (used during restore to check compatibility)
- `db_checksum_sha256` — SHA-256 of the `zfaktury.db` file inside the archive
- `db_size_bytes` — size of the database file before compression
- `documents_count` — number of document files included
- `documents_size_bytes` — total size of document files before compression
- `archive_size_bytes` — final `.tar.gz` size (written to DB record after archiving)

#### Naming Convention

```
zfaktury-backup-{ISO8601}.tar.gz
```

Timestamp uses dashes instead of colons for filesystem compatibility: `2026-03-13T14-30-00`.

### Sync Strategy

Built-in sync is out of scope — it introduces distributed systems complexity (conflict resolution, partial transfers, authentication) that is not justified for a single-user desktop/server app.

Instead, ZFaktury supports external sync by allowing `ZFAKTURY_DATA_DIR` to point at a synced folder.

#### Instance Lock

To prevent data corruption from concurrent writes (e.g., two machines syncing the same folder), ZFaktury acquires an exclusive file lock at startup:

```go
// Acquired in cmd startup, held for process lifetime
lockFile := filepath.Join(dataDir, ".zfaktury.lock")
f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_RDWR, 0600)
if err != nil {
    return fmt.Errorf("opening lock file: %w", err)
}
if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
    return fmt.Errorf("another ZFaktury instance is using this data directory: %w", err)
}
```

If the lock cannot be acquired, the application exits with a clear error message.

#### Journal Mode Configuration

SQLite WAL mode is faster but does not work reliably on network filesystems (NFS, SMB, FUSE-based mounts). For users pointing `ZFAKTURY_DATA_DIR` at a network mount:

```toml
[database]
journal_mode = "wal"      # Default — fast, local disk only
# journal_mode = "delete"  # Use for network filesystems (NAS, Syncthing, Nextcloud)
```

The journal mode is set at database open time via `PRAGMA journal_mode`.

### Config

```toml
[backup]
# Destination directory for backup archives.
# Default: {DataDir}/backups
destination = ""

# Cron expression for automatic backups (empty = disabled).
# Examples: "0 2 * * *" (daily at 2:00), "0 3 * * 0" (weekly Sunday at 3:00)
schedule = ""

# Number of backups to retain. Oldest are deleted when exceeded.
# 0 = keep all backups (no automatic deletion).
retention_count = 10
```

Config struct addition:

```go
type BackupConfig struct {
    Destination    string `toml:"destination"`
    Schedule       string `toml:"schedule"`
    RetentionCount int    `toml:"retention_count"`
}
```

When `destination` is empty, backups are stored in `{DataDir}/backups/`. The directory is created automatically on first backup.

### Database Migration (024)

```sql
-- +goose Up
CREATE TABLE backup_history (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    filename        TEXT    NOT NULL,
    file_path       TEXT    NOT NULL,
    file_size_bytes INTEGER NOT NULL DEFAULT 0,
    db_size_bytes   INTEGER NOT NULL DEFAULT 0,
    docs_count      INTEGER NOT NULL DEFAULT 0,
    docs_size_bytes INTEGER NOT NULL DEFAULT 0,
    db_checksum     TEXT    NOT NULL DEFAULT '',
    app_version     TEXT    NOT NULL DEFAULT '',
    migration_ver   INTEGER NOT NULL DEFAULT 0,
    status          TEXT    NOT NULL DEFAULT 'running',  -- running, completed, failed
    error_message   TEXT    NOT NULL DEFAULT '',
    started_at      TEXT    NOT NULL,
    completed_at    TEXT,
    created_at      TEXT    NOT NULL DEFAULT (datetime('now')),
    deleted_at      TEXT
);

-- +goose Down
DROP TABLE backup_history;
```

### Domain

```go
type BackupStatus string

const (
    BackupStatusRunning   BackupStatus = "running"
    BackupStatusCompleted BackupStatus = "completed"
    BackupStatusFailed    BackupStatus = "failed"
)

type Backup struct {
    ID             int64
    Filename       string
    FilePath       string
    FileSizeBytes  int64
    DBSizeBytes    int64
    DocsCount      int
    DocsSizeBytes  int64
    DBChecksum     string
    AppVersion     string
    MigrationVer   int
    Status         BackupStatus
    ErrorMessage   string
    StartedAt      time.Time
    CompletedAt    *time.Time
    CreatedAt      time.Time
}
```

### Repository Interface

```go
type BackupRepository interface {
    Create(ctx context.Context, b *domain.Backup) (int64, error)
    Update(ctx context.Context, b *domain.Backup) error
    GetByID(ctx context.Context, id int64) (*domain.Backup, error)
    List(ctx context.Context, limit, offset int) ([]domain.Backup, int, error)
    SoftDelete(ctx context.Context, id int64) error
}
```

### Service

```go
type BackupService struct {
    repo    repository.BackupRepository
    db      *sql.DB
    cfg     config.BackupConfig
    dataDir string
}
```

Key methods:

- `CreateBackup(ctx) (*domain.Backup, error)` — creates snapshot, bundles archive, records in DB
- `RestoreBackup(ctx, archivePath string) error` — validates archive, checks migration compatibility, replaces DB and documents
- `ListBackups(ctx, limit, offset int) ([]domain.Backup, int, error)` — paginated history
- `GetBackup(ctx, id int64) (*domain.Backup, error)` — detail
- `DeleteBackup(ctx, id int64) error` — soft-delete record, remove archive file
- `GetBackupFilePath(ctx, id int64) (string, error)` — returns path for download
- `ApplyRetention(ctx) error` — delete oldest backups exceeding `retention_count`
- `GetStatus(ctx) (*BackupStatusInfo, error)` — last backup time, next scheduled, running state

#### Backup Flow

1. Insert `backup_history` row with `status = running`
2. `VACUUM INTO` to temp file
3. Compute SHA-256 of the snapshot
4. Create `.tar.gz` with snapshot + documents directory + `backup-meta.json`
5. Move archive to destination directory
6. Update row with `status = completed`, sizes, checksum
7. Run retention cleanup
8. On any error: update row with `status = failed`, `error_message`

#### Restore Flow

1. Verify archive integrity (extract `backup-meta.json`, verify DB checksum)
2. Check `migration_version` — refuse restore if archive version is newer than current app
3. Stop accepting new requests (or require CLI-only restore)
4. Replace database file and documents directory from archive
5. Run pending migrations if archive version is older
6. Restart application

### API Endpoints

All endpoints under `/api/v1/backups`.

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/backups` | Trigger a new backup |
| `GET` | `/api/v1/backups` | List backup history (paginated) |
| `GET` | `/api/v1/backups/status` | Current backup status (last backup, next scheduled, running) |
| `GET` | `/api/v1/backups/{id}` | Backup detail |
| `DELETE` | `/api/v1/backups/{id}` | Delete backup (soft-delete record, remove file) |
| `GET` | `/api/v1/backups/{id}/download` | Download backup archive |

#### Request/Response DTOs

```go
// POST /api/v1/backups — no request body needed
// Response: BackupDTO (201 Created)

type BackupDTO struct {
    ID             int64   `json:"id"`
    Filename       string  `json:"filename"`
    FileSizeBytes  int64   `json:"file_size_bytes"`
    DBSizeBytes    int64   `json:"db_size_bytes"`
    DocsCount      int     `json:"docs_count"`
    DocsSizeBytes  int64   `json:"docs_size_bytes"`
    Status         string  `json:"status"`
    ErrorMessage   string  `json:"error_message,omitempty"`
    AppVersion     string  `json:"app_version"`
    MigrationVer   int     `json:"migration_ver"`
    StartedAt      string  `json:"started_at"`
    CompletedAt    *string `json:"completed_at,omitempty"`
    CreatedAt      string  `json:"created_at"`
}

type BackupStatusDTO struct {
    LastBackup    *BackupDTO `json:"last_backup"`
    NextScheduled *string    `json:"next_scheduled,omitempty"`
    IsRunning     bool       `json:"is_running"`
    Schedule      string     `json:"schedule"`
}
```

The download endpoint streams the file with `Content-Disposition: attachment` and `Content-Type: application/gzip`.

### CLI Commands

#### `zfaktury backup`

Creates a backup immediately, printing progress to stdout.

```
$ zfaktury backup
Creating backup...
  Database snapshot: 1.2 MB
  Documents: 42 files, 50.3 MB
  Archive: backups/zfaktury-backup-2026-03-13T14-30-00.tar.gz (30.1 MB)
Backup completed successfully.
```

#### `zfaktury restore <file>`

Restores from a backup archive. Requires confirmation.

```
$ zfaktury restore backups/zfaktury-backup-2026-03-13T14-30-00.tar.gz
Backup info:
  Version: 0.12.0 (migration: 24)
  Database: 1.2 MB (checksum OK)
  Documents: 42 files, 50.3 MB

WARNING: This will replace all current data.
Type 'yes' to confirm: yes

Restoring database... done
Restoring documents... done
Running migrations... 0 pending
Restore completed successfully.
```

Flags:
- `--force` — skip confirmation prompt (for scripted usage)

### Scheduled Backups

When `backup.schedule` is set to a cron expression, the backup service starts a goroutine that:

1. Parses the cron expression using `github.com/robfig/cron/v3`
2. Sleeps until the next scheduled time
3. Runs `CreateBackup` (which includes retention cleanup)
4. Logs result via `slog`

The scheduler starts in `serve.go` alongside the HTTP server and stops on shutdown signal.

### Frontend

New page at `/settings/backup` accessible from the settings navigation.

#### Layout

1. **Status card** — last backup time, next scheduled backup, backup size
2. **Trigger button** — "Vytvorit zalohu" (Create backup), disabled while running, shows progress
3. **History table** — columns: date, filename, size, status, actions (download, delete)
4. **Sync info** — informational panel explaining how to set up external sync with `ZFAKTURY_DATA_DIR`

#### Components

- `BackupStatusCard.svelte` — displays current backup status, auto-refreshes while running
- Reuses existing table and button components from the shared library

#### API Client Addition

```typescript
// Backup types
export interface Backup {
  id: number;
  filename: string;
  file_size_bytes: number;
  db_size_bytes: number;
  docs_count: number;
  docs_size_bytes: number;
  status: 'running' | 'completed' | 'failed';
  error_message?: string;
  app_version: string;
  migration_ver: number;
  started_at: string;
  completed_at?: string;
  created_at: string;
}

export interface BackupStatus {
  last_backup: Backup | null;
  next_scheduled: string | null;
  is_running: boolean;
  schedule: string;
}

// Methods on ApiClient
createBackup(): Promise<Backup>
listBackups(limit?: number, offset?: number): Promise<PaginatedResponse<Backup>>
getBackup(id: number): Promise<Backup>
deleteBackup(id: number): Promise<void>
getBackupStatus(): Promise<BackupStatus>
downloadBackup(id: number): Promise<Blob>
```

## Implementation Phases

### Phase 1 — Core Backup & Restore

- Domain struct, repository, migration (024)
- Backup service with `VACUUM INTO`, tar.gz archiving, metadata
- CLI commands: `backup`, `restore`
- Config parsing for `[backup]` section

### Phase 2 — API & Frontend

- HTTP handler with all endpoints
- Frontend backup page with status, trigger, history
- Download functionality

### Phase 3 — Scheduling & Retention

- Cron-based scheduler goroutine
- Retention policy enforcement
- Sync documentation and instance lock

## Dependencies

| Purpose | Package |
|---------|---------|
| Cron scheduler | `github.com/robfig/cron/v3` |
| Archive | `archive/tar`, `compress/gzip` (stdlib) |
| Checksum | `crypto/sha256` (stdlib) |
| File lock | `syscall` (stdlib) |

## Open Questions

1. **Maximum backup size** — should there be a configurable limit to prevent filling the disk? Could warn when destination has less than 2x the expected backup size free.
2. **Restore via API** — the current design limits restore to CLI only (requires process restart). A future enhancement could support API-triggered restore with automatic server restart.
3. **Encryption** — backup archives contain sensitive financial data. Should encryption (e.g., age or AES-256-GCM with a user-provided passphrase) be supported in Phase 1 or deferred?
