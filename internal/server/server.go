// Package server gère le cycle de vie du serveur HTTP :
// initialisation, enregistrement des routes, démarrage et arrêt propre.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/AaronDAmin/magenta/internal/config"
	"github.com/AaronDAmin/magenta/internal/media"
	"github.com/AaronDAmin/magenta/internal/stream"
)

// Server encapsule le serveur HTTP et ses dépendances.
type Server struct {
	http          *http.Server     // serveur HTTP standard de la bibliothèque Go
	cfg           *config.Config
	log           *slog.Logger
	router        *http.ServeMux   // routeur interne
	index         *media.Index     // index des médias
	streamHandler *stream.Handler  // handler de streaming
}

// New crée et configure un nouveau Server prêt à démarrer.
// Il enregistre toutes les routes et les middlewares au moment de la construction.
func New(cfg *config.Config, log *slog.Logger, index *media.Index) *Server {
	mux := http.NewServeMux()

	s := &Server{
		cfg:           cfg,
		log:           log,
		router:        mux,
		index:         index,
		streamHandler: stream.NewHandler(log, index),
	}

	// Enregistrement des routes — chaque groupe de routes est isolé
	// dans sa propre méthode pour garder le code lisible.
	s.registerRoutes()

	// Construction du serveur HTTP avec les timeouts configurés.
	// L'adresse d'écoute combine l'hôte et le port issus de la config,
	// ce qui restreint le serveur à une interface réseau précise.
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	s.http = &http.Server{
		Addr: addr,
		// Le routeur est enveloppé dans le middleware de logging
		// pour tracer automatiquement toutes les requêtes entrantes.
		Handler: s.loggingMiddleware(mux),

		// ReadTimeout : durée maximale pour lire l'intégralité de la requête.
		// Protège contre les clients lents qui gardent la connexion ouverte.
		ReadTimeout: time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,

		// WriteTimeout : durée maximale pour écrire la réponse.
		// Mis à 0 dans config.yaml pour le streaming (pas de limite de temps).
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,

		// Injecte notre logger dans le serveur HTTP pour les erreurs internes.
		ErrorLog: slog.NewLogLogger(log.Handler(), slog.LevelError),
	}

	return s
}

// Start démarre le serveur HTTP en arrière-plan dans une goroutine.
// Il retourne immédiatement pour ne pas bloquer l'appelant.
// Utiliser Shutdown() pour arrêter proprement le serveur.
func (s *Server) Start() {
	s.log.Info("serveur démarré",
		slog.String("adresse", s.http.Addr),
	)

	// ListenAndServe bloque jusqu'à l'arrêt du serveur.
	// On le lance dans une goroutine pour rendre la main à main().
	// ErrServerClosed est retourné normalement lors d'un Shutdown() — ce n'est pas une erreur.
	go func() {
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Error("erreur fatale du serveur HTTP", slog.String("erreur", err.Error()))
		}
	}()
}

// Shutdown arrête le serveur proprement en attendant la fin des connexions actives.
// Le context passé en paramètre permet de définir un délai d'attente maximum.
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("arrêt du serveur en cours...")

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("erreur lors de l'arrêt du serveur : %w", err)
	}

	s.log.Info("serveur arrêté proprement")
	return nil
}

// registerRoutes enregistre toutes les routes HTTP du serveur.
// Cette méthode est le point central de définition de l'API.
func (s *Server) registerRoutes() {
	// Route de santé — permet de vérifier que le serveur répond correctement.
	// Accessible depuis un navigateur ou avec : curl http://<ip>:8080/health
	s.router.HandleFunc("GET /health", s.handleHealth)

	// Route de l'index des médias — retourne la liste complète des fichiers indexés
	s.router.HandleFunc("GET /api/media", s.handleMediaList)

	// Route pour récupérer l'épisode suivant dans le même dossier
	s.router.HandleFunc("GET /api/media/{id}/next", s.handleMediaNext)

	// Route pour générer une playlist M3U d'un dossier
	s.router.HandleFunc("GET /api/playlist", s.handlePlaylist)

	// Route de streaming — diffuse un fichier multimédia par son ID
	// Supporte les range requests pour le scrubbing vidéo dans VLC
	s.router.HandleFunc("GET /stream/{id}", s.streamHandler.ServeStream)

	// TODO (Étape 5) : routes d'administration des appareils
	// s.router.HandleFunc("GET /admin/devices", s.handleDeviceList)
	// s.router.HandleFunc("POST /admin/devices", s.handleDeviceAdd)
	// s.router.HandleFunc("DELETE /admin/devices/{id}", s.handleDeviceRevoke)
}
