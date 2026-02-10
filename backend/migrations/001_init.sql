CREATE TABLE IF NOT EXISTS keywords (
    id         SERIAL PRIMARY KEY,
    value      TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS matched_certificates (
    id             SERIAL PRIMARY KEY,
    serial_number  TEXT        NOT NULL,
    common_name    TEXT        NOT NULL,
    sans           TEXT[]      NOT NULL DEFAULT '{}',
    issuer         TEXT        NOT NULL,
    not_before     TIMESTAMPTZ,
    not_after      TIMESTAMPTZ,
    keyword_id     INTEGER     NOT NULL REFERENCES keywords(id) ON DELETE CASCADE,
    matched_domain TEXT        NOT NULL,
    ct_log_index   BIGINT      NOT NULL,
    discovered_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(serial_number, keyword_id)
);

CREATE INDEX IF NOT EXISTS idx_matched_certs_keyword
    ON matched_certificates(keyword_id);
CREATE INDEX IF NOT EXISTS idx_matched_certs_discovered
    ON matched_certificates(discovered_at DESC);

CREATE TABLE IF NOT EXISTS monitor_state (
    id                     INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    last_processed_index   BIGINT  NOT NULL DEFAULT 0,
    last_tree_size         BIGINT  NOT NULL DEFAULT 0,
    last_run_at            TIMESTAMPTZ,
    total_processed        BIGINT  NOT NULL DEFAULT 0,
    certs_in_last_cycle    INTEGER NOT NULL DEFAULT 0,
    matches_in_last_cycle  INTEGER NOT NULL DEFAULT 0,
    is_running             BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO monitor_state (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
