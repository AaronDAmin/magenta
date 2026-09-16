// Package media gère le scan, l'indexation et la surveillance des fichiers multimédias.
package media

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Media représente un fichier multimédia indexé par le serveur.
type Media struct {
	// ID est un identifiant unique généré à partir du chemin absolu du fichier.
	// Il est stable tant que le fichier ne change pas d'emplacement.
	ID string `json:"id"`

	// Name est le nom du fichier avec son extension (ex: "Film.2024.mkv").
	Name string `json:"name"`

	// Path est le chemin absolu du fichier sur le disque.
	// Ce champ n'est PAS exposé dans l'API publique pour des raisons de sécurité.
	Path string `json:"-"`

	// Folder est le chemin du dossier parent contenant ce fichier.
	// Utile pour regrouper les épisodes d'une série ou les films d'une collection.
	Folder string `json:"folder"`

	// Size est la taille du fichier en octets.
	Size int64 `json:"size"`

	// Extension est l'extension du fichier sans le point (ex: "mkv", "mp4").
	Extension string `json:"extension"`

	// ModifiedAt est la date de dernière modification du fichier.
	ModifiedAt time.Time `json:"modified_at"`
}

// NewMedia crée un Media à partir du chemin complet d'un fichier et de ses métadonnées.
// L'ID est calculé en hashant le chemin avec SHA256.
func NewMedia(path, name, ext, folder string, size int64, modTime time.Time) *Media {
	return &Media{
		ID:         generateID(path),
		Name:       name,
		Path:       path,
		Folder:     folder,
		Size:       size,
		Extension:  ext,
		ModifiedAt: modTime,
	}
}

// generateID génère un identifiant unique stable à partir du chemin du fichier.
// Utilise SHA256 pour garantir l'unicité et l'absence de collision.
func generateID(path string) string {
	hash := sha256.Sum256([]byte(path))
	// On prend les 16 premiers octets (32 caractères hex) pour garder un ID lisible
	return hex.EncodeToString(hash[:16])
}
