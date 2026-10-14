CREATE TABLE outbox_events (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    event_id VARCHAR(36) NOT NULL UNIQUE,
    topic VARCHAR(255) NOT NULL,
    msg_key VARBINARY(512) NOT NULL,
    payload LONGBLOB NOT NULL,
    headers JSON NOT NULL,
    status VARCHAR(16) NOT NULL,
    attempt INT NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    available_at DATETIME(6) NOT NULL,
    locked_until DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL,
    sent_at DATETIME(6) NULL,
    INDEX outbox_claim_idx (status, available_at, locked_until, id),
    INDEX outbox_key_order_idx (msg_key, id, status)
);