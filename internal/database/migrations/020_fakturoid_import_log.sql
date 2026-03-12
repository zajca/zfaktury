-- +goose Up
CREATE TABLE fakturoid_import_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fakturoid_entity_type TEXT NOT NULL,
    fakturoid_id INTEGER NOT NULL,
    local_entity_type TEXT NOT NULL,
    local_id INTEGER NOT NULL,
    imported_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE(fakturoid_entity_type, fakturoid_id)
);

-- +goose Down
DROP TABLE IF EXISTS fakturoid_import_log;
