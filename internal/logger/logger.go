// Package logger fournit un logger structuré centralisé pour l'ensemble du projet.
// Il s'appuie sur le package log/slog de la bibliothèque standard Go (>=1.21)
// afin de produire des logs en format JSON ou texte selon l'environnement.
package logger

import (
	"log/slog"
	"os"
)

// Level représente le niveau de verbosité du logger.
type Level = slog.Level

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// New crée et retourne un nouveau logger structuré.
// Si jsonFormat est true, les logs sont émis en JSON (recommandé en production).
// Sinon, un format texte lisible est utilisé (recommandé en développement).
func New(level Level, jsonFormat bool) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: level,
		// Ajoute le nom du fichier source et la ligne dans chaque log (utile pour le debug)
		AddSource: level == LevelDebug,
	}

	var handler slog.Handler
	if jsonFormat {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
