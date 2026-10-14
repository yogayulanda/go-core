CREATE TABLE dbo.outbox_events (
    id BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
    event_id NVARCHAR(36) NOT NULL UNIQUE,
    topic NVARCHAR(255) NOT NULL,
    msg_key VARBINARY(512) NOT NULL,
    payload VARBINARY(MAX) NOT NULL,
    headers NVARCHAR(MAX) NOT NULL,
    status NVARCHAR(16) NOT NULL,
    attempt INT NOT NULL CONSTRAINT DF_outbox_attempt DEFAULT 0,
    last_error NVARCHAR(MAX) NULL,
    available_at DATETIME2(6) NOT NULL,
    locked_until DATETIME2(6) NULL,
    created_at DATETIME2(6) NOT NULL,
    sent_at DATETIME2(6) NULL
);
CREATE INDEX outbox_claim_idx ON dbo.outbox_events (status, available_at, locked_until, id);
CREATE INDEX outbox_key_order_idx ON dbo.outbox_events (msg_key, id, status);