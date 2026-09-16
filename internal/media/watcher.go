// Package media — surveillance des changements de fichiers avec fsnotify.
package media

import (
	"log/slog"
	"path/filepath"
	"time"

	"github.com/AaronDAmin/magenta/internal/config"
	"github.com/fsnotify/fsnotify"
)

// Watcher surveille les répertoires configurés et met à jour l'index automatiquement
// lors de l'ajout, modification ou suppression de fichiers.
type Watcher struct {
	cfg     *config.Config
	log     *slog.Logger
	index   *Index
	scanner *Scanner
	watcher *fsnotify.Watcher
	done    chan struct{} // signal pour arrêter le watcher proprement
}

// NewWatcher crée un nouveau watcher pour surveiller les changements de fichiers.
func NewWatcher(cfg *config.Config, log *slog.Logger, index *Index, scanner *Scanner) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		cfg:     cfg,
		log:     log,
		index:   index,
		scanner: scanner,
		watcher: fw,
		done:    make(chan struct{}),
	}, nil
}

// Start démarre la surveillance des répertoires configurés.
// Cette méthode bloque et doit être appelée dans une goroutine.
func (w *Watcher) Start() error {
	// Ajout de tous les répertoires configurés au watcher
	for _, dir := range w.cfg.Media.Dirs {
		if err := w.watcher.Add(dir); err != nil {
			w.log.Error("impossible de surveiller le répertoire",
				slog.String("chemin", dir),
				slog.String("erreur", err.Error()),
			)
			continue
		}
		w.log.Info("surveillance active", slog.String("répertoire", dir))
	}

	// Boucle d'écoute des événements
	// Debounce : on attend 500ms sans événement avant de traiter (évite le spam sur copie de masse)
	var debounceTimer *time.Timer
	pendingEvents := make(map[string]fsnotify.Op)

	processPendingEvents := func() {
		for path, op := range pendingEvents {
			w.handleEvent(path, op)
		}
		pendingEvents = make(map[string]fsnotify.Op)
	}

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil // watcher fermé
			}

			// Accumulation des événements dans la map (le dernier op gagne)
			pendingEvents[event.Name] = event.Op

			// Reset du timer de debounce
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceTimer = time.AfterFunc(500*time.Millisecond, processPendingEvents)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil // watcher fermé
			}
			w.log.Error("erreur du watcher", slog.String("erreur", err.Error()))

		case <-w.done:
			w.log.Info("arrêt de la surveillance des fichiers")
			return nil
		}
	}
}

// Stop arrête proprement le watcher.
func (w *Watcher) Stop() error {
	close(w.done)
	return w.watcher.Close()
}

// handleEvent traite un événement fsnotify et met à jour l'index en conséquence.
func (w *Watcher) handleEvent(path string, op fsnotify.Op) {
	// Normalisation du chemin
	path = filepath.Clean(path)

	switch {
	case op&fsnotify.Create == fsnotify.Create:
		// Nouveau fichier créé
		if w.scanner.ScanFile(path) {
			w.log.Info("média ajouté", slog.String("chemin", path))
		}

	case op&fsnotify.Write == fsnotify.Write:
		// Fichier modifié (potentiellement en cours d'écriture)
		// On re-scanne pour mettre à jour la taille et la date de modification
		if w.scanner.ScanFile(path) {
			w.log.Debug("média mis à jour", slog.String("chemin", path))
		}

	case op&fsnotify.Remove == fsnotify.Remove:
		// Fichier supprimé
		if w.index.Remove(path) {
			w.log.Info("média supprimé", slog.String("chemin", path))
		}

	case op&fsnotify.Rename == fsnotify.Rename:
		// Fichier renommé ou déplacé
		// On le supprime de l'index (fsnotify enverra un Create si le nouveau chemin est surveillé)
		if w.index.Remove(path) {
			w.log.Info("média renommé/déplacé", slog.String("chemin", path))
		}
	}
}
