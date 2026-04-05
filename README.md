# fauxtes-server

A lightweight [Standard Notes](https://standardnotes.com/) sync server written in Go with a SQLite backend. Designed as a single-binary, self-hostable alternative to the official [Standard Notes server](https://github.com/standardnotes/server).

## Features

- Standard Notes sync protocol (V004) compatible
- User registration, sign-in, and session management
- Item sync with conflict detection and pagination
- PKCE authentication flow
- SQLite storage (zero-config, single-file database)
- Optional TLS support
- 164 tests

## Requirements

- Go 1.24+
- C compiler (GCC or Clang) for the SQLite driver (`mattn/go-sqlite3`)

## Quick Start

```bash
# Build
go build -o fauxtes ./cmd/fauxtes

# Run
./fauxtes

# Or use make
make run
```

The server starts on `:8080` with a `fauxtes.db` SQLite database in the current directory.

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `FAUXTES_ADDR` | `:8080` | Listen address (host:port) |
| `FAUXTES_DB` | `fauxtes.db` | SQLite database file path |
| `FAUXTES_TLS_CERT` | _(none)_ | Path to TLS certificate file (PEM) |
| `FAUXTES_TLS_KEY` | _(none)_ | Path to TLS private key file (PEM) |

When both `FAUXTES_TLS_CERT` and `FAUXTES_TLS_KEY` are set, the server uses HTTPS. Otherwise it serves plain HTTP.

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/healthcheck` | No | Health check |
| `POST` | `/auth` | No | Register a new account |
| `GET` | `/auth/params` | No | Get key derivation params for an email |
| `POST` | `/auth/sign_in` | No | Sign in with credentials |
| `POST` | `/session/token` | No | Refresh an expired session |
| `DELETE` | `/session` | Yes | Sign out (delete session) |
| `POST` | `/items/sync` | Yes | Sync items (create, update, retrieve) |

## Connecting the Standard Notes App

Point the Standard Notes app at your server URL. The app requires HTTPS for custom servers.

### TLS with a Reverse Proxy (Recommended)

Run fauxtes on plain HTTP behind a reverse proxy that handles TLS. Example with [Caddy](https://caddyserver.com/):

```
notes.example.com {
    reverse_proxy localhost:8080
}
```

Caddy automatically provisions and renews Let's Encrypt certificates.

### TLS with Self-Signed Certificates

For local development or LAN use:

```bash
# Generate a self-signed cert (replace IP with your server's address)
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
  -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/CN=fauxtes" -addext "subjectAltName=IP:YOUR_IP_HERE"

# Run with TLS
FAUXTES_TLS_CERT=cert.pem FAUXTES_TLS_KEY=key.pem ./fauxtes
```

Note: Android apps do not trust user-installed certificates by default. For mobile use, a publicly-trusted certificate (via Let's Encrypt or similar) is required.

## Usage Examples

```bash
# Register
curl -s http://localhost:8080/auth -X POST \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"hashed_pw","pw_nonce":"nonce","version":"004","api":"20200115"}'

# Get key params
curl -s "http://localhost:8080/auth/params?email=user@example.com"

# Sign in
curl -s http://localhost:8080/auth/sign_in -X POST \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"hashed_pw","api":"20200115","code_verifier":"verifier"}'

# Sync items (use access_token from sign-in response)
curl -s http://localhost:8080/items/sync -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ACCESS_TOKEN" \
  -d '{"items":[],"sync_token":null,"limit":150}'
```

## Docker

```bash
# Build
docker build -t fauxtes .

# Run
docker run -p 8080:8080 -v fauxtes-data:/data fauxtes

# Run with TLS
docker run -p 8080:8080 \
  -v fauxtes-data:/data \
  -v ./cert.pem:/cert.pem:ro \
  -v ./key.pem:/key.pem:ro \
  -e FAUXTES_TLS_CERT=/cert.pem \
  -e FAUXTES_TLS_KEY=/key.pem \
  fauxtes
```

The database is stored at `/data/fauxtes.db` inside the container. Mount a volume to `/data` to persist it.

## Development

```bash
# Run tests
make test

# Build
make build

# Clean build artifacts
make clean
```

### Project Structure

```
cmd/fauxtes/           Entry point
internal/
  auth/                Authentication (service, handlers, middleware, crypto)
  domain/              Shared types (User, Session, Item, SyncToken)
  server/              HTTP server wiring and integration tests
  storage/sqlite/      SQLite repository implementations and migrations
  sync/                Sync protocol (service, handlers, GetItems, SaveItems)
```

## Scope and Limitations

This server implements the minimum Standard Notes functionality needed for single-user sync:

- Registration and sign-in (protocol V004)
- Session management with token refresh
- Item sync with conflict detection and cursor-based pagination
- Pseudo key params for non-existent users (prevents email enumeration)
- Account locking after failed sign-in attempts

Not implemented:

- Shared vaults
- File uploads
- Messages and notifications
- Subscription/payment features
- Multi-user roles and permissions
- Postgres backend (planned)

## License

See [LICENSE](LICENSE) for details.
