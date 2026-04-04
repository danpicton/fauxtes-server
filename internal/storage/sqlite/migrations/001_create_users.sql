-- +goose Up
CREATE TABLE users (
    uuid                     TEXT PRIMARY KEY,
    email                    VARCHAR(255),
    encrypted_password       VARCHAR(255) NOT NULL,
    pw_nonce                 VARCHAR(255),
    pw_salt                  VARCHAR(255),
    pw_cost                  INTEGER,
    pw_key_size              INTEGER,
    pw_alg                   VARCHAR(255),
    pw_func                  VARCHAR(255),
    encrypted_server_key     VARCHAR(255),
    server_encryption_version INTEGER NOT NULL DEFAULT 0,
    version                  VARCHAR(255),
    kp_created               VARCHAR(255),
    kp_origination           VARCHAR(255),
    locked_until             DATETIME,
    num_failed_attempts      INTEGER,
    created_at               DATETIME NOT NULL,
    updated_at               DATETIME NOT NULL
);
CREATE UNIQUE INDEX idx_users_email ON users(email);

-- +goose Down
DROP TABLE users;
