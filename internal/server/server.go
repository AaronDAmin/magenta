// Package server gère le cycle de vie du serveur HTTP Magenta.
package server

import (
	"context"
	cryptotls "crypto/tls"
	"embed"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/AaronDAmin/magenta/internal/auth"
	"github.com/AaronDAmin/magenta/internal/config"
	"github.com/AaronDAmin/magenta/internal/media"
	"github.com/AaronDAmin/magenta/internal/stream"
)

// Server encapsule le serveur HTTP et ses dépendances.
type Server struct {
	http          *http.Server
	https         *http.Server
	cfg           *config.Config
	log           *slog.Logger
	router        *http.ServeMux
	index         *media.Index
	streamHandler *stream.Handler
	userStore     *auth.UserStore
	authHandler   *auth.Handler
	webFS         embed.FS
}

// New crée et configure un nouveau Server.
func New(cfg *config.Config, log *slog.Logger, index *media.Index, userStore *auth.UserStore, webFS embed.FS) *Server {
	mux := http.NewServeMux()

	s := &Server{
		cfg:           cfg,
		log:           log,
		router:        mux,
		index:         index,
		streamHandler: stream.NewHandler(log, index),
		userStore:     userStore,
		authHandler:   auth.NewHandler(userStore, log),
		webFS:         webFS,
	}

	s.registerRoutes()

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	s.http = &http.Server{
		Addr:         addr,
		Handler:      s.loggingMiddleware(mux),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,
		ErrorLog:     slog.NewLogLogger(log.Handler(), slog.LevelError),
	}

	return s
}

// EnableTLS configure le serveur HTTPS.
func (s *Server) EnableTLS(tlsConfig *cryptotls.Config, port int) {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, port)
	s.https = &http.Server{
		Addr:        addr,
		Handler:     s.loggingMiddleware(s.router),
		TLSConfig:   tlsConfig,
		ReadTimeout: time.Duration(s.cfg.Server.ReadTimeoutSec) * time.Second,
		ErrorLog:    slog.NewLogLogger(s.log.Handler(), slog.LevelError),
	}
}

// Start démarre le serveur HTTP (et HTTPS si configuré).
func (s *Server) Start() {
	s.log.Info("serveur démarré", slog.String("adresse", s.http.Addr))

	go func() {
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("erreur fatale HTTP", slog.String("erreur", err.Error()))
		}
	}()

	if s.https != nil {
		s.log.Info("serveur HTTPS démarré", slog.String("adresse", s.https.Addr))
		go func() {
			if err := s.https.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				s.log.Error("erreur fatale HTTPS", slog.String("erreur", err.Error()))
			}
		}()
	}
}

// Shutdown arrête proprement le serveur.
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("arrêt du serveur...")

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("arrêt HTTP : %w", err)
	}

	if s.https != nil {
		if err := s.https.Shutdown(ctx); err != nil {
			return fmt.Errorf("arrêt HTTPS : %w", err)
		}
	}

	s.log.Info("serveur arrêté proprement")
	return nil
}

// registerRoutes enregistre toutes les routes HTTP.
func (s *Server) registerRoutes() {
	authMiddleware := auth.Middleware(s.userStore, s.log)

	// ── Santé ──────────────────────────────────────────────────────────────
	s.router.HandleFunc("GET /health", s.handleHealth)

	// ── Auth publique (pas de token requis) ────────────────────────────────
	s.router.HandleFunc("POST /api/auth/register", s.authHandler.HandleRegister)
	s.router.HandleFunc("POST /api/auth/login", s.authHandler.HandleLogin)
	s.router.HandleFunc("GET /api/auth/status", s.authHandler.HandleStatus)

	// ── API médias (token requis) ───────────────────────────────────────────
	s.router.Handle("GET /api/media", authMiddleware(http.HandlerFunc(s.handleMediaList)))
	s.router.Handle("GET /api/media/{id}/next", authMiddleware(http.HandlerFunc(s.handleMediaNext)))
	s.router.Handle("GET /api/folders", authMiddleware(http.HandlerFunc(s.handleFolders)))
	s.router.Handle("GET /api/playlist", authMiddleware(http.HandlerFunc(s.handlePlaylist)))

	// ── Streaming (token requis) ───────────────────────────────────────────
	s.router.Handle("GET /stream/{id}", authMiddleware(http.HandlerFunc(s.streamHandler.ServeStream)))
	s.router.Handle("GET /stream/{id}/vlc", authMiddleware(http.HandlerFunc(s.handleVLC)))

	// ── Admin (localhost uniquement, géré dans le middleware) ──────────────
	s.router.HandleFunc("GET /admin/users", s.authHandler.HandleAdminListUsers)
	s.router.HandleFunc("POST /admin/users/{id}/approve", s.authHandler.HandleAdminApprove)
	s.router.HandleFunc("POST /admin/users/{id}/revoke", s.authHandler.HandleAdminRevoke)
	s.router.HandleFunc("DELETE /admin/users/{id}", s.authHandler.HandleAdminDelete)

	// ── Interface web ──────────────────────────────────────────────────────
	s.router.Handle("/", s.serveWeb(s.webFS))
}
