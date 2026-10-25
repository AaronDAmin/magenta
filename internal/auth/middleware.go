// Package auth — middleware d'authentification HTTP.
package auth

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// logRateLimiter empêche le spam de logs pour les mêmes erreurs répétées.
// Chaque clé (ip+token_prefix) n'est loguée qu'une seule fois par minute.
type logRateLimiter struct {
	mu      sync.Mutex
	seen    map[string]time.Time
	ttl     time.Duration
}

func newLogRateLimiter() *logRateLimiter {
	rl := &logRateLimiter{
		seen: make(map[string]time.Time),
		ttl:  time.Minute,
	}
	// Nettoyage périodique pour éviter les fuites mémoire
	go func() {
		for range time.Tick(5 * time.Minute) {
			rl.mu.Lock()
			cutoff := time.Now().Add(-rl.ttl)
			for k, t := range rl.seen {
				if t.Before(cutoff) {
					delete(rl.seen, k)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

// allow retourne true si ce log doit être émis (première fois dans la fenêtre TTL).
func (rl *logRateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if last, ok := rl.seen[key]; ok && time.Since(last) < rl.ttl {
		return false // déjà loggé récemment
	}
	rl.seen[key] = time.Now()
	return true
}

// Middleware crée un middleware d'authentification basé sur UserStore.
// Routes publiques (sans token) :
//   - /health
//   - /api/auth/* (login, register, status)
//   - /static/*
//   - /login, /register
//
// Routes admin (localhost uniquement) :
//   - /admin/*
//
// Toutes les autres routes nécessitent un token valide.
func Middleware(store *UserStore, log *slog.Logger) func(http.Handler) http.Handler {
	rl := newLogRateLimiter()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Routes publiques
			if path == "/health" ||
				path == "/login" ||
				path == "/register" ||
				strings.HasPrefix(path, "/api/auth/") ||
				strings.HasPrefix(path, "/static/") {
				next.ServeHTTP(w, r)
				return
			}

			// Routes admin : localhost uniquement, pas de token requis
			if strings.HasPrefix(path, "/admin") {
				if !isLocalhost(r.RemoteAddr) {
					log.Warn("tentative d'accès admin hors localhost",
						slog.String("ip", r.RemoteAddr),
						slog.String("path", path),
					)
					http.Error(w, "Accès refusé : admin accessible uniquement depuis localhost", http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Toutes les autres routes : token requis
			token := extractToken(r)
			if token == "" {
				// Rate-limit : une seule entrée de log par IP par minute
				key := "notoken:" + clientIP(r.RemoteAddr)
				if rl.allow(key) {
					log.Warn("requête sans token",
						slog.String("ip", r.RemoteAddr),
						slog.String("path", path),
					)
				}
				if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/stream/") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"Token d'authentification manquant"}`))
					return
				}
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			user := store.GetByToken(token)
			if user == nil {
				// Rate-limit : une seule entrée de log par token invalide par minute.
				// Évite le spam quand le navigateur vidéo retente le stream en boucle.
				key := "badtoken:" + token[:min(8, len(token))]
				if rl.allow(key) {
					log.Warn("token invalide ou révoqué",
						slog.String("ip", r.RemoteAddr),
						slog.String("path", path),
						slog.String("token_prefix", token[:min(8, len(token))]),
					)
				}
				if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/stream/") {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"Token invalide ou révoqué"}`))
					return
				}
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			log.Debug("authentification réussie",
				slog.String("username", user.Username),
				slog.String("ip", r.RemoteAddr),
			)

			next.ServeHTTP(w, r)
		})
	}
}

// extractToken extrait le token depuis (par priorité) :
// 1. Cookie magenta_token (interface web)
// 2. Query parameter ?token= (VLC, URLs directes)
// 3. Header Authorization: Bearer <token> (clients API)
func extractToken(r *http.Request) string {
	if cookie, err := r.Cookie("magenta_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}

	if auth := r.Header.Get("Authorization"); auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	return ""
}

// isLocalhost vérifie si l'adresse est localhost (127.0.0.1, ::1).
func isLocalhost(remoteAddr string) bool {
	host := clientIP(remoteAddr)
	return host == "127.0.0.1" || host == "::1" || host == "localhost"
}

// clientIP extrait l'adresse IP depuis "ip:port" ou "[ipv6]:port".
func clientIP(remoteAddr string) string {
	host := remoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		host = remoteAddr[:idx]
	}
	return strings.Trim(host, "[]")
}

// min retourne le minimum de deux entiers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
