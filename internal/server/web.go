// Package server — handlers pour l'interface web.
package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// WebFS contient les fichiers de l'interface web embarqués.
// Il est défini dans le package principal et passé au serveur.
type WebFS struct {
	FS embed.FS
}

// serveWeb retourne un handler pour servir les fichiers statiques et les pages HTML.
func (s *Server) serveWeb(webFS embed.FS) http.Handler {
	// Extraction du sous-système de fichiers web/
	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		s.log.Error("impossible de charger le système de fichiers web", "erreur", err.Error())
		return http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Vérification de l'authentification via cookie pour les pages protégées
		if path != "/login" && !strings.HasPrefix(path, "/static/") {
			token := extractTokenFromCookie(r)
			if token == "" {
				// Pas de token dans le cookie, redirige vers login
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Vérification de la validité du token
			device := s.authStore.GetByToken(token)
			if device == nil {
				// Token invalide, redirige vers login
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
		}

		// Routage des pages
		switch path {
		case "/":
			// Page d'accueil = bibliothèque
			serveFile(w, r, webRoot, "library.html")
		case "/login":
			serveFile(w, r, webRoot, "login.html")
		case "/player":
			serveFile(w, r, webRoot, "player.html")
		default:
			// Fichiers statiques ou 404
			http.FileServer(http.FS(webRoot)).ServeHTTP(w, r)
		}
	})
}

// serveFile sert un fichier HTML depuis le système de fichiers embarqué.
func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, filename string) {
	content, err := fs.ReadFile(fsys, filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

// extractTokenFromCookie extrait le token depuis le cookie magenta_token.
func extractTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("magenta_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}
