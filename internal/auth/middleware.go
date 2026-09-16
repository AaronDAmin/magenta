// Package auth — middleware d'authentification HTTP.
package auth

import (
	"log/slog"
	"net/http"
	"strings"
)

// Middleware crée un middleware d'authentification qui vérifie le token sur toutes les routes protégées.
// Les routes suivantes sont exclues de l'authentification :
// - /health (santé du serveur)
// - /admin/* (gestion des appareils, accessible uniquement depuis localhost)
func Middleware(store *Store, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Routes publiques (pas d'authentification requise)
			if r.URL.Path == "/health" || 
			   r.URL.Path == "/login" || 
			   strings.HasPrefix(r.URL.Path, "/static/") {
				next.ServeHTTP(w, r)
				return
			}

			// Routes admin : accessibles uniquement depuis localhost
			if strings.HasPrefix(r.URL.Path, "/admin/") {
				if !isLocalhost(r.RemoteAddr) {
					log.Warn("tentative d'accès admin depuis une IP non-localhost",
						slog.String("ip", r.RemoteAddr),
						slog.String("path", r.URL.Path),
					)
					http.Error(w, "Accès refusé : les routes admin ne sont accessibles que depuis localhost", http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Toutes les autres routes nécessitent un token valide
			token := extractToken(r)
			if token == "" {
				log.Warn("requête sans token",
					slog.String("ip", r.RemoteAddr),
					slog.String("path", r.URL.Path),
				)
				http.Error(w, "Token d'authentification manquant", http.StatusUnauthorized)
				return
			}

			// Vérification du token dans le store
			device := store.GetByToken(token)
			if device == nil {
				log.Warn("token invalide",
					slog.String("ip", r.RemoteAddr),
					slog.String("path", r.URL.Path),
					slog.String("token", token[:min(8, len(token))]+"..."), // log partiel pour debug
				)
				http.Error(w, "Token d'authentification invalide", http.StatusUnauthorized)
				return
			}

			// Token valide, on laisse passer
			log.Debug("authentification réussie",
				slog.String("appareil", device.Name),
				slog.String("ip", r.RemoteAddr),
			)

			next.ServeHTTP(w, r)
		})
	}
}

// extractToken extrait le token depuis les sources suivantes (par ordre de priorité) :
// 1. Cookie : magenta_token (pour l'interface web)
// 2. Query parameter : ?token=xyz (pour VLC et les URLs directes)
// 3. Header Authorization : Bearer xyz (pour les clients API)
func extractToken(r *http.Request) string {
	// 1. Cookie (prioritaire pour l'interface web)
	cookie, err := r.Cookie("magenta_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 2. Query parameter (pour VLC)
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}

	// 3. Header Authorization (format : "Bearer <token>")
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	return ""
}

// isLocalhost vérifie si une adresse IP correspond à localhost.
func isLocalhost(remoteAddr string) bool {
	// RemoteAddr format : "ip:port" ou "[ipv6]:port"
	host := remoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		host = remoteAddr[:idx]
	}

	// Suppression des crochets IPv6
	host = strings.Trim(host, "[]")

	// Vérification des adresses localhost
	return host == "127.0.0.1" || host == "::1" || host == "localhost"
}

// min retourne le minimum de deux entiers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
