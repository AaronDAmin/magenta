// Package main — point d'entrée du serveur multimédia Magenta.
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AaronDAmin/magenta/internal/auth"
	"github.com/AaronDAmin/magenta/internal/config"
	"github.com/AaronDAmin/magenta/internal/logger"
	"github.com/AaronDAmin/magenta/internal/mdns"
	"github.com/AaronDAmin/magenta/internal/media"
	"github.com/AaronDAmin/magenta/internal/network"
	"github.com/AaronDAmin/magenta/internal/server"
	internaltls "github.com/AaronDAmin/magenta/internal/tls"
)

//go:embed all:web
var webFS embed.FS

var version = "dev"

func main() {
	configPath := flag.String("config", "config.yaml", "chemin vers le fichier de configuration")
	showVersion := flag.Bool("version", false, "afficher la version et quitter")
	flag.Parse()

	if *showVersion {
		fmt.Printf("magenta %s\n", version)
		os.Exit(0)
	}

	// ── Configuration ──────────────────────────────────────────────────────
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERREUR : configuration : %v\n", err)
		os.Exit(1)
	}

	// ── Logger ─────────────────────────────────────────────────────────────
	logLevel := parseLogLevel(cfg.Log.Level)
	log := logger.New(logLevel, cfg.Log.JSON)

	log.Info("magenta démarrage",
		slog.String("version", version),
		slog.String("config", *configPath),
	)

	// ── Store utilisateurs ─────────────────────────────────────────────────
	userStore, err := auth.NewUserStore("users.json", log)
	if err != nil {
		log.Error("impossible de charger le store utilisateurs", slog.String("erreur", err.Error()))
		os.Exit(1)
	}
	log.Info("store utilisateurs initialisé", slog.Int("utilisateurs", userStore.Count()))

	// ── Index médias + scanner ─────────────────────────────────────────────
	index := media.NewIndex()
	scanner := media.NewScanner(cfg, log, index)

	go func() {
		if err := scanner.ScanAll(); err != nil {
			log.Error("erreur scan initial", slog.String("erreur", err.Error()))
		}
	}()

	// ── Watcher fichiers ───────────────────────────────────────────────────
	watcher, err := media.NewWatcher(cfg, log, index, scanner)
	if err != nil {
		log.Error("impossible de créer le watcher", slog.String("erreur", err.Error()))
		os.Exit(1)
	}
	go func() {
		if err := watcher.Start(); err != nil {
			log.Error("erreur watcher", slog.String("erreur", err.Error()))
		}
	}()

	// ── mDNS ───────────────────────────────────────────────────────────────
	var mdnsServer *mdns.Server
	if cfg.MDNS.Enabled {
		name := cfg.MDNS.Name
		if name == "" {
			name = "magenta"
		}
		mdnsServer, err = mdns.New(log, name, cfg.Server.Port)
		if err != nil {
			log.Warn("mDNS indisponible", slog.String("erreur", err.Error()))
		}
	}

	// ── Signaux système ────────────────────────────────────────────────────
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ── Serveur HTTP ───────────────────────────────────────────────────────
	srv := server.New(cfg, log, index, userStore, webFS)

	if cfg.TLS.Enabled {
		certCfg := internaltls.CertConfig{
			CertFile: cfg.TLS.CertFile,
			KeyFile:  cfg.TLS.KeyFile,
		}
		hosts := []string{"magenta.local", cfg.Server.Host}
		tlsConfig, err := internaltls.LoadOrGenerate(certCfg, log, hosts)
		if err != nil {
			log.Error("impossible d'initialiser TLS", slog.String("erreur", err.Error()))
			os.Exit(1)
		}
		srv.EnableTLS(tlsConfig, cfg.TLS.Port)
		log.Info("HTTPS activé", slog.Int("port", cfg.TLS.Port))
	}

	srv.Start()

	// ── Banner ─────────────────────────────────────────────────────────────
	mdnsName := ""
	if cfg.MDNS.Enabled {
		mdnsName = cfg.MDNS.Name
	}
	network.PrintBanner(network.AccessInfo{
		Port: cfg.Server.Port,
		Name: mdnsName,
	})

	// ── Attente signal arrêt ───────────────────────────────────────────────
	<-ctx.Done()
	log.Info("signal d'arrêt reçu...")

	if err := watcher.Stop(); err != nil {
		log.Error("erreur arrêt watcher", slog.String("erreur", err.Error()))
	}
	if mdnsServer != nil {
		mdnsServer.Stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("erreur arrêt serveur", slog.String("erreur", err.Error()))
		os.Exit(1)
	}

	log.Info("magenta arrêté proprement")
}

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
