-- +goose Up

-- One row per attempt to send a generated filing through a data box. The XML
-- itself is not copied here: xml_sha256 identifies the exact bytes that were sent.
CREATE TABLE filing_submissions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    -- Kind of the document: vat-return, control-statement, vies-summary,
    -- income-tax-return or social-insurance.
    document_kind TEXT NOT NULL
        CHECK (document_kind IN ('vat-return', 'control-statement', 'vies-summary', 'income-tax-return', 'social-insurance')),
    document_id INTEGER NOT NULL,
    channel TEXT NOT NULL CHECK (channel IN ('isds')),
    environment TEXT NOT NULL CHECK (environment IN ('test', 'production')),
    recipient_box_id TEXT NOT NULL,
    recipient_name TEXT NOT NULL,
    -- ISDS message id; empty until CreateMessage answered.
    dm_id TEXT NOT NULL DEFAULT '',
    -- 1 when the user typed dm_id (resolve), 0 when it came from our own CreateMessage.
    dm_id_attached INTEGER NOT NULL DEFAULT 0 CHECK (dm_id_attached IN (0, 1)),
    dm_annotation TEXT NOT NULL DEFAULT '',
    file_name TEXT NOT NULL,
    -- Lowercase hex sha256 of the sent XML bytes.
    xml_sha256 TEXT NOT NULL CHECK (length(xml_sha256) = 64),
    -- sending: the request is about to leave or is in flight.
    -- unknown: the outcome of CreateMessage could not be determined (transport failure).
    -- sent: ISDS accepted the message, delivery is not confirmed yet.
    -- delivered: dmMessageStatus 4 or higher except 8.
    -- undeliverable: dmMessageStatus 8.
    -- failed: ISDS definitively refused the message.
    status TEXT NOT NULL
        CHECK (status IN ('sending', 'unknown', 'sent', 'delivered', 'undeliverable', 'failed')),
    -- Last dmMessageStatus received as is; NULL until a delivery query ran.
    dm_message_status INTEGER,
    sent_at TEXT,
    delivered_at TEXT,
    -- Signed sent message (ZFO) fetched after delivery.
    receipt_zfo BLOB,
    error_text TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    deleted_at TEXT
);

CREATE INDEX idx_filing_submissions_document ON filing_submissions (document_kind, document_id);

-- The same bytes are never sent twice to the same environment unless the earlier
-- attempt definitively failed or was undeliverable. A row stuck in sending or
-- unknown also blocks, because the message may have left.
CREATE UNIQUE INDEX idx_filing_submissions_one_send
    ON filing_submissions (document_kind, document_id, environment, xml_sha256)
    WHERE deleted_at IS NULL AND status IN ('sending', 'unknown', 'sent', 'delivered');

-- +goose Down

DROP INDEX IF EXISTS idx_filing_submissions_one_send;
DROP INDEX IF EXISTS idx_filing_submissions_document;
DROP TABLE IF EXISTS filing_submissions;
