// Package main est le point d'entrée du serveur multimédia Magenta.
// Il orchestre le chargement de la configuration, l'initialisation du logger,
// et le démarrage du serveur HTTP.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/AaronDAmin/magenta/internal/config"
	"github.com/AaronDAmin/magenta/internal/logger"
)

// version est injectée au moment du build via -ldflags.
// Exemple : go build -ldflags "-X main.version=1.0.0"
var version = "dev"

func main() {
	// --- Parsing des arguments CLI ---

	// Chemin vers le fichier de configuration (peut être surchargé via flag)
	configPath := flag.String("config", "config.yaml", "chemin vers le fichier de configuration")
	// Afficher la version et quitter
	showVersion := flag.Bool("version", false, "afficher la version et quitter")
	flag.Parse()

	if *showVersion {
		fmt.Printf("magenta %s\n", version)
		os.Exit(0)
	}

	// --- Chargement de la configuration ---
	cfg, err := config.Load(*configPath)
	if err != nil {
		// Le logger n'est pas encore initialisé ici, on utilise stderr directement
		fmt.Fprintf(os.Stderr, "ERREUR : impossible de charger la configuration : %v\n", err)
		os.Exit(1)
	}

	// --- Initialisation du logger ---
	logLevel := parseLogLevel(cfg.Log.Level)
	log := logger.New(logLevel, cfg.Log.JSON)

	log.Info("magenta démarrage",
		slog.String("version", version),
		slog.String("config", *configPath),
	)
	log.Debug("configuration chargée", slog.String("config", cfg.String()))

	// --- Gestion des signaux système pour un arrêt propre ---
	// On écoute SIGINT (Ctrl+C) et SIGTERM (signal d'arrêt système)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- TODO (Étape 2) : initialisation et démarrage du serveur HTTP ---
	// server := server.New(cfg, log)
	// server.Start(ctx)

	// Attente du signal d'arrêt
	<-ctx.Done()
	log.Info("signal d'arrêt reçu, extinction en cours...")

	// TODO (Étape 2) : appel server.Shutdown()

	log.Info("magenta arrêté proprement")
}

// parseLogLevel convertit une chaîne de caractères en niveau slog.
// Valeurs acceptées : "debug", "info", "warn", "error".
// Retourne LevelInfo si la valeur est inconnue.
func parseLogLevel(level string) logger.Level {
	switch level {
	case "debug":
		return logger.LevelDebug
	case "warn":
		return logger.LevelWarn
	case "error":
		return logger.LevelError
	default:
		return logger.LevelInfo
	}
}
