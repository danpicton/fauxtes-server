-- +goose Up
CREATE TABLE sessions (
    uuid                 TEXT PRIMARY KEY,
    user_uuid            VARCHAR(255),
    private_identifier   VARCHAR(36),
    hashed_access_token  VARCHAR(255) NOT NULL,
    hashed_refresh_token VARCHAR(255) NOT NULL,
    access_expiration    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    refresh_expiration   DATETIME NOT NULL,
    api_version          VARCHAR(255),
    user_agent           TEXT,
    readonly_access      INTEGER NOT NULL DEFAULT 0,
    version              INTEGER DEFAULT 1,
    application          VARCHAR(255),
    snjs                 VARCHAR(255),
    created_at           DATETIME NOT NULL,
    updated_at           DATETIME NOT NULL
);
CREATE INDEX idx_sessions_user_uuid ON sessions(user_uuid);
CREATE INDEX idx_sessions_access_token ON sessions(hashed_access_token);
CREATE INDEX idx_sessions_refresh_token ON sessions(hashed_refresh_token);
CREATE INDEX idx_sessions_updated_at ON sessions(updated_at);

-- +goose Down
DROP TABLE sessions;
