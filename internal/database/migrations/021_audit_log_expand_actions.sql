-- +goose Up

-- Drop the restrictive CHECK constraint on audit_log.action to allow new action types:
-- activate, deactivate, set, set_bulk, generate_xml, mark_filed, recalculate
-- SQLite does not support ALTER TABLE DROP CONSTRAINT, so we recreate the table.

CREATE TABLE audit_log_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    old_values TEXT,
    new_values TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

INSERT INTO audit_log_new (id, entity_type, entity_id, action, old_values, new_values, created_at)
    SELECT id, entity_type, entity_id, action, old_values, new_values, created_at FROM audit_log;

DROP TABLE audit_log;
ALTER TABLE audit_log_new RENAME TO audit_log;

CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);

-- +goose Down

-- Restore the original CHECK constraint (entries with non-standard actions will be lost).
CREATE TABLE audit_log_old (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('create', 'update', 'delete')),
    old_values TEXT,
    new_values TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

INSERT INTO audit_log_old (id, entity_type, entity_id, action, old_values, new_values, created_at)
    SELECT id, entity_type, entity_id, action, old_values, new_values, created_at
    FROM audit_log WHERE action IN ('create', 'update', 'delete');

DROP TABLE audit_log;
ALTER TABLE audit_log_old RENAME TO audit_log;

CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_created_at ON audit_log(created_at);
