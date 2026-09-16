// Package stream gère le streaming HTTP des fichiers multimédias.
// Il implémente le support complet des range requests (RFC 7233) indispensable
// pour la lecture vidéo avec seek/scrubbing dans VLC et autres lecteurs.
package stream

import (
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"os"

	"github.com/AaronDAmin/magenta/internal/media"
)

// Handler gère les requêtes de streaming de fichiers multimédias.
type Handler struct {
	log   *slog.Logger
	index *media.Index
}

// NewHandler crée un nouveau handler de streaming.
func NewHandler(log *slog.Logger, index *media.Index) *Handler {
	return &Handler{
		log:   log,
		index: index,
	}
}

// ServeStream gère le streaming d'un fichier multimédia identifié par son ID.
// Cette méthode utilise http.ServeContent qui gère automatiquement :
// - Les range requests (RFC 7233) pour le scrubbing vidéo
// - La détection du Content-Type
// - Les en-têtes Accept-Ranges, Content-Range, etc.
// - La gestion des erreurs 206 Partial Content et 416 Range Not Satisfiable
//
// Exemple d'URL : GET /stream/a3f5e8d2...
func (h *Handler) ServeStream(w http.ResponseWriter, r *http.Request) {
	// Extraction de l'ID depuis le path parameter
	id := r.PathValue("id")
	if id == "" {
		h.respondError(w, http.StatusBadRequest, "ID manquant dans l'URL")
		return
	}

	// Récupération du média depuis l'index
	m := h.index.Get(id)
	if m == nil {
		h.log.Warn("média introuvable",
			slog.String("id", id),
			slog.String("ip", r.RemoteAddr),
		)
		h.respondError(w, http.StatusNotFound, "Média introuvable")
		return
	}

	// Ouverture du fichier sur le disque
	file, err := os.Open(m.Path)
	if err != nil {
		h.log.Error("impossible d'ouvrir le fichier",
			slog.String("id", id),
			slog.String("chemin", m.Path),
			slog.String("erreur", err.Error()),
		)
		h.respondError(w, http.StatusInternalServerError, "Impossible de lire le fichier")
		return
	}
	defer file.Close()

	// Récupération des métadonnées du fichier
	stat, err := file.Stat()
	if err != nil {
		h.log.Error("impossible de lire les métadonnées",
			slog.String("id", id),
			slog.String("erreur", err.Error()),
		)
		h.respondError(w, http.StatusInternalServerError, "Erreur de lecture des métadonnées")
		return
	}

	// Détermination du Content-Type à partir de l'extension
	contentType := detectContentType(m.Extension)
	w.Header().Set("Content-Type", contentType)

	// En-têtes supplémentaires pour optimiser le streaming
	// Accept-Ranges indique au client que les range requests sont supportées
	w.Header().Set("Accept-Ranges", "bytes")

	// Cache-Control : les médias sont immutables une fois indexés
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	// Content-Disposition : suggère le nom du fichier au client
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, m.Name))

	h.log.Info("streaming démarré",
		slog.String("id", id),
		slog.String("nom", m.Name),
		slog.Int64("taille", m.Size),
		slog.String("ip", r.RemoteAddr),
		slog.String("range", r.Header.Get("Range")),
	)

	// http.ServeContent gère automatiquement :
	// - Les range requests (status 206)
	// - Les en-têtes Content-Range
	// - La validation If-Modified-Since / If-None-Match
	// - Les erreurs 416 Range Not Satisfiable
	http.ServeContent(w, r, m.Name, stat.ModTime(), file)
}

// detectContentType retourne le MIME type approprié pour une extension de fichier.
// Utilise la base de données MIME standard de Go, avec des fallbacks pour les formats vidéo courants.
func detectContentType(ext string) string {
	// Tentative de détection via la bibliothèque mime standard
	mimeType := mime.TypeByExtension("." + ext)
	if mimeType != "" {
		return mimeType
	}

	// Fallback manuel pour les extensions vidéo courantes non reconnues
	// ou reconnues avec un mauvais type MIME
	switch ext {
	case "mp4":
		return "video/mp4"
	case "mkv":
		return "video/x-matroska"
	case "avi":
		return "video/x-msvideo"
	case "mov":
		return "video/quicktime"
	case "wmv":
		return "video/x-ms-wmv"
	case "flv":
		return "video/x-flv"
	case "webm":
		return "video/webm"
	case "m4v":
		return "video/x-m4v"
	default:
		// Fallback ultime : application/octet-stream (fichier binaire générique)
		return "application/octet-stream"
	}
}

// respondError envoie une réponse d'erreur JSON au client.
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error": "%s"}`, message)
}
