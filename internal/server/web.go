// Package server — handlers pour l'interface web embarquée.
package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// serveWeb retourne un handler pour les pages HTML et fichiers statiques.
func (s *Server) serveWeb(webFS embed.FS) http.Handler {
	webRoot, err := fs.Sub(webFS, "web")
	if err != nil {
		s.log.Error("impossible de charger le FS web", "erreur", err.Error())
		return http.NotFoundHandler()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Fichiers statiques : servir directement sans auth
		if strings.HasPrefix(path, "/static/") {
			http.FileServer(http.FS(webRoot)).ServeHTTP(w, r)
			return
		}

		// Pages publiques (pas de redirection auth)
		if path == "/login" || path == "/register" {
			// Si déjà connecté, redirige vers la bibliothèque
			if token := extractTokenFromCookie(r); token != "" {
				if user := s.userStore.GetByToken(token); user != nil {
					http.Redirect(w, r, "/", http.StatusSeeOther)
					return
				}
			}
			switch path {
			case "/login":
				serveFile(w, r, webRoot, "login.html")
			case "/register":
				serveFile(w, r, webRoot, "register.html")
			}
			return
		}

		// Page admin : localhost uniquement
		if path == "/admin" || strings.HasPrefix(path, "/admin/") {
			serveFile(w, r, webRoot, "admin.html")
			return
		}

		// Pages protégées : vérification du token
		token := extractTokenFromCookie(r)
		if token == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if user := s.userStore.GetByToken(token); user == nil {
			// Token invalide ou révoqué
			http.SetCookie(w, &http.Cookie{
				Name:   "magenta_token",
				Value:  "",
				MaxAge: -1,
				Path:   "/",
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Routage des pages protégées
		switch path {
		case "/", "/library":
			serveFile(w, r, webRoot, "library.html")
		case "/player":
			serveFile(w, r, webRoot, "player.html")
		default:
			http.FileServer(http.FS(webRoot)).ServeHTTP(w, r)
		}
	})
}

func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, filename string) {
	content, err := fs.ReadFile(fsys, filename)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(content)
}

func extractTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie("magenta_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}
