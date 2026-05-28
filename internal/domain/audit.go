package domain

import "time"

// AuditLogEntry represents a single audit trail record for entity changes.
//
// CompanyID is populated from the request context when the entry is recorded
// under a per-company route. System-level actions that have no active company
// leave this nil so the per-company filter on the audit-log page can ignore
// them.
type AuditLogEntry struct {
	ID         int64
	CompanyID  *int64
	EntityType string
	EntityID   int64
	Action     string
	OldValues  string
	NewValues  string
	CreatedAt  time.Time
}

// AuditLogFilter defines filtering options for listing audit log entries.
type AuditLogFilter struct {
	EntityType string
	EntityID   *int64
	Action     string
	From       time.Time
	To         time.Time
	// CompanyID, when non-nil, restricts results to audit log entries whose
	// company_id matches the given id. Entries with NULL company_id (system
	// events, cross-company actions) are excluded by this filter.
	CompanyID *int64
	Limit     int
	Offset    int
}
