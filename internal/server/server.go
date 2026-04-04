package server

import (
	"database/sql"
	"net/http"

	"github.com/danpicton/fauxtes-server/internal/auth"
	"github.com/danpicton/fauxtes-server/internal/storage/sqlite"
	syncpkg "github.com/danpicton/fauxtes-server/internal/sync"
)

type Config struct {
	Addr       string
	DBPath     string
	TLSCert    string
	TLSKey     string
	AuthConfig auth.ServiceConfig
}

func DefaultConfig() Config {
	return Config{
		Addr:       ":8080",
		DBPath:     "fauxtes.db",
		AuthConfig: auth.DefaultServiceConfig(),
	}
}

type Server struct {
	DB  *sql.DB
	Mux *http.ServeMux
}

func New(cfg Config) (*Server, error) {
	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}
	if err := sqlite.Migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	return NewWithDB(db, cfg)
}

func NewWithDB(db *sql.DB, cfg Config) (*Server, error) {
	userRepo := sqlite.NewUserRepo(db)
	sessionRepo := sqlite.NewSessionRepo(db)
	itemRepo := sqlite.NewItemRepo(db)

	authService := auth.NewService(userRepo, sessionRepo, cfg.AuthConfig)
	authHandler := auth.NewHandler(authService)

	syncService := syncpkg.NewService(itemRepo)
	syncHandler := syncpkg.NewHandler(syncService)

	authMW := auth.AuthMiddleware(sessionRepo)

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("GET /healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Auth routes (unauthenticated)
	mux.HandleFunc("POST /auth", authHandler.Register)
	mux.HandleFunc("GET /auth/params", authHandler.GetParams)
	mux.HandleFunc("POST /auth/sign_in", authHandler.SignIn)
	mux.HandleFunc("POST /session/token", authHandler.RefreshSession)

	// Auth routes (authenticated)
	mux.Handle("DELETE /session", authMW(http.HandlerFunc(authHandler.SignOut)))

	// Sync routes (authenticated)
	mux.Handle("POST /items/sync", authMW(http.HandlerFunc(syncHandler.Sync)))

	return &Server{DB: db, Mux: mux}, nil
}
