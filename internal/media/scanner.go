// Package media — scanner de fichiers multimédias.
package media

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/AaronDAmin/magenta/internal/config"
)

// Scanner parcourt récursivement les dossiers configurés et indexe les fichiers multimédias.
type Scanner struct {
	cfg   *config.Config
	log   *slog.Logger
	index *Index
}

// NewScanner crée un nouveau scanner avec la configuration fournie.
func NewScanner(cfg *config.Config, log *slog.Logger, index *Index) *Scanner {
	return &Scanner{
		cfg:   cfg,
		log:   log,
		index: index,
	}
}

// ScanAll parcourt tous les répertoires configurés et indexe tous les médias trouvés.
// Cette opération peut prendre du temps sur des gros volumes de fichiers.
// Elle est conçue pour tourner en arrière-plan sans bloquer le serveur.
func (s *Scanner) ScanAll() error {
	s.log.Info("début du scan des médias", slog.Int("répertoires", len(s.cfg.Media.Dirs)))

	totalFiles := 0
	totalErrors := 0

	// Créer une map des extensions autorisées pour un lookup O(1)
	allowedExts := make(map[string]bool)
	for _, ext := range s.cfg.Media.Extensions {
		// Normalisation : toujours en minuscule, sans point
		allowedExts[strings.ToLower(strings.TrimPrefix(ext, "."))] = true
	}

	// Scan de chaque répertoire configuré
	for _, dir := range s.cfg.Media.Dirs {
		s.log.Debug("scan du répertoire", slog.String("chemin", dir))

		// Vérification que le répertoire existe avant de scanner
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			s.log.Warn("répertoire introuvable, ignoré", slog.String("chemin", dir))
			continue
		}

		// Scan récursif avec filepath.WalkDir (plus rapide que Walk)
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			// Gestion des erreurs de lecture (permissions, fichier supprimé pendant le scan, etc.)
			if err != nil {
				s.log.Warn("erreur lors de l'accès au fichier",
					slog.String("chemin", path),
					slog.String("erreur", err.Error()),
				)
				totalErrors++
				return nil // Continue le scan malgré l'erreur
			}

			// Ignorer les répertoires, on ne traite que les fichiers
			if d.IsDir() {
				return nil
			}

			// Vérifier l'extension du fichier
			ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
			if !allowedExts[ext] {
				return nil // Extension non autorisée, on ignore
			}

			// Récupérer les métadonnées du fichier
			info, err := d.Info()
			if err != nil {
				s.log.Warn("impossible de lire les métadonnées",
					slog.String("chemin", path),
					slog.String("erreur", err.Error()),
				)
				totalErrors++
				return nil
			}

			// Créer l'objet Media et l'ajouter à l'index
			media := NewMedia(
				path,
				filepath.Base(path),
				ext,
				filepath.Dir(path), // dossier parent
				info.Size(),
				info.ModTime(),
			)
			s.index.Add(media)
			totalFiles++

			return nil
		})

		if err != nil {
			s.log.Error("erreur fatale lors du scan",
				slog.String("répertoire", dir),
				slog.String("erreur", err.Error()),
			)
			return fmt.Errorf("scan de %q : %w", dir, err)
		}
	}

	s.log.Info("scan terminé",
		slog.Int("fichiers_indexés", totalFiles),
		slog.Int("erreurs", totalErrors),
	)

	return nil
}

// ScanFile scanne un seul fichier et l'ajoute à l'index s'il correspond aux critères.
// Cette méthode est utilisée par fsnotify lors de l'ajout/modification d'un fichier.
// Retourne true si le fichier a été indexé.
func (s *Scanner) ScanFile(path string) bool {
	// Vérifier que le fichier existe
	info, err := os.Stat(path)
	if err != nil {
		s.log.Debug("fichier introuvable, ignoré",
			slog.String("chemin", path),
			slog.String("erreur", err.Error()),
		)
		return false
	}

	// Ignorer les répertoires
	if info.IsDir() {
		return false
	}

	// Vérifier l'extension
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	isAllowed := false
	for _, allowedExt := range s.cfg.Media.Extensions {
		if ext == strings.ToLower(allowedExt) {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return false
	}

	// Créer et indexer le média
	media := NewMedia(
		path,
		filepath.Base(path),
		ext,
		filepath.Dir(path), // dossier parent
		info.Size(),
		info.ModTime(),
	)
	s.index.Add(media)

	s.log.Debug("fichier indexé",
		slog.String("nom", media.Name),
		slog.String("id", media.ID),
	)

	return true
}
