-- +goose Up
CREATE TABLE items (
    uuid                    TEXT PRIMARY KEY,
    user_uuid               VARCHAR(36) NOT NULL,
    content                 TEXT,
    content_type            VARCHAR(255),
    content_size            INTEGER,
    enc_item_key            TEXT,
    auth_hash               VARCHAR(255),
    items_key_id            VARCHAR(255),
    duplicate_of            VARCHAR(36),
    last_edited_by          VARCHAR(36),
    updated_with_session    VARCHAR(36),
    deleted                 INTEGER NOT NULL DEFAULT 0,
    shared_vault_uuid       VARCHAR(36),
    key_system_identifier   VARCHAR(36),
    created_at              DATETIME NOT NULL,
    updated_at              DATETIME NOT NULL,
    created_at_timestamp    BIGINT NOT NULL,
    updated_at_timestamp    BIGINT NOT NULL
);
CREATE INDEX idx_items_user_uuid ON items(user_uuid);
CREATE INDEX idx_items_user_updated ON items(user_uuid, updated_at_timestamp);
CREATE INDEX idx_items_user_content_type ON items(user_uuid, content_type);
CREATE INDEX idx_items_user_deleted ON items(user_uuid, deleted);

-- +goose Down
DROP TABLE items;
