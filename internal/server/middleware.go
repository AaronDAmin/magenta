// Package server — middlewares HTTP.
// Les middlewares sont des fonctions qui interceptent et enrichissent les requêtes HTTP
// avant qu'elles n'atteignent les handlers, et après que ceux-ci aient répondu.
package server

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter est un wrapper autour de http.ResponseWriter qui capture
// le code de statut HTTP retourné par le handler.
//
// Par défaut, http.ResponseWriter ne permet pas de lire le statut après l'envoi,
// mais on en a besoin pour le logging. Ce wrapper intercepte l'appel à WriteHeader()
// pour capturer le code avant de le transmettre à l'implémentation réelle.
type responseWriter struct {
	http.ResponseWriter          // intègre tous les comportements par défaut
	statusCode          int      // code de statut capturé (200, 404, 401, etc.)
	written             bool     // indique si WriteHeader() a déjà été appelé
}

// newResponseWriter crée un responseWriter autour d'un ResponseWriter standard.
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK, // par défaut, Go envoie 200 si non spécifié
		written:        false,
	}
}

// WriteHeader capture le code de statut avant de le transmettre à l'implémentation réelle.
// Cette méthode surcharge celle de http.ResponseWriter.
func (rw *responseWriter) WriteHeader(code int) {
	// On ne capture qu'une seule fois (le premier appel à WriteHeader compte)
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

// Write appelle WriteHeader implicitement si ça n'a pas encore été fait.
// C'est le comportement standard de http.ResponseWriter, on le reproduit ici.
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// loggingMiddleware est un middleware qui logge automatiquement chaque requête HTTP.
// Il enregistre :
// - la méthode HTTP (GET, POST, etc.)
// - le chemin demandé
// - l'adresse IP du client
// - le code de réponse (200, 404, 401, etc.)
// - la durée de traitement de la requête
//
// Ce middleware enveloppe le handler suivant et s'exécute pour toutes les routes.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture du moment où la requête arrive
		start := time.Now()

		// Wrap du ResponseWriter pour capturer le code de statut
		wrappedWriter := newResponseWriter(w)

		// Appel du handler suivant (la route réelle ou le prochain middleware)
		next.ServeHTTP(wrappedWriter, r)

		// Calcul de la durée de traitement
		duration := time.Since(start)

		// Log structuré de la requête complète
		s.log.Info("requête HTTP",
			slog.String("méthode", r.Method),
			slog.String("chemin", r.URL.Path),
			slog.String("ip", r.RemoteAddr),
			slog.Int("statut", wrappedWriter.statusCode),
			slog.Duration("durée", duration),
		)
	})
}
