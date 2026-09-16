// Package server — handlers HTTP.
// Ce fichier contient tous les gestionnaires de routes HTTP du serveur.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// handleHealth est la route de santé du serveur.
// Elle retourne un statut 200 OK avec un JSON minimal pour confirmer
// que le serveur est actif et répond correctement aux requêtes.
//
// Exemple de réponse :
//
//	{
//	  "status": "ok",
//	  "service": "magenta"
//	}
//
// Cette route est utile pour :
// - Vérifier manuellement que le serveur fonctionne (depuis un navigateur)
// - Monitorer le serveur depuis un outil externe (Prometheus, Uptime Kuma, etc.)
// - Tester la connectivité réseau depuis un appareil client
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Préparation de la réponse JSON
	response := map[string]string{
		"status":  "ok",
		"service": "magenta",
	}

	// Définition de l'en-tête Content-Type
	w.Header().Set("Content-Type", "application/json")

	// Encodage et envoi de la réponse
	// Si l'encodage échoue, json.NewEncoder retourne une erreur mais on ne peut
	// plus modifier les headers HTTP à ce stade (déjà envoyés). On logge l'erreur.
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error("erreur lors de l'encodage JSON de /health",
			"erreur", err.Error(),
		)
	}
}

// handleMediaList retourne la liste complète des fichiers multimédias indexés.
//
// Exemple de réponse :
//
//	{
//	  "total": 42,
//	  "items": [
//	    {
//	      "id": "a3f5e8d2...",
//	      "name": "Film.2024.1080p.mkv",
//	      "size": 2147483648,
//	      "extension": "mkv",
//	      "modified_at": "2026-09-15T18:30:00Z"
//	    },
//	    ...
//	  ]
//	}
func (s *Server) handleMediaList(w http.ResponseWriter, r *http.Request) {
	// Récupération de tous les médias depuis l'index
	allMedia := s.index.All()

	// Construction de la réponse JSON
	response := map[string]interface{}{
		"total": len(allMedia),
		"items": allMedia,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error("erreur lors de l'encodage JSON de /api/media",
			"erreur", err.Error(),
		)
	}
}

// handleMediaNext retourne le média suivant dans le même dossier (ordre alphabétique).
// Utile pour la lecture automatique des épisodes d'une série.
//
// Exemple de requête : GET /api/media/abc123/next
//
// Réponse si un épisode suivant existe :
//
//	{
//	  "id": "def456",
//	  "name": "Episode_02.mkv",
//	  "folder": "D:/Videos/Series/Season1",
//	  ...
//	}
//
// Réponse si c'est le dernier épisode : 404 Not Found
func (s *Server) handleMediaNext(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "ID manquant", http.StatusBadRequest)
		return
	}

	next := s.index.GetNext(id)
	if next == nil {
		http.Error(w, "Aucun média suivant trouvé", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(next); err != nil {
		s.log.Error("erreur lors de l'encodage JSON de /api/media/{id}/next",
			"erreur", err.Error(),
		)
	}
}

// handlePlaylist génère une playlist M3U pour tous les fichiers d'un dossier.
// Compatible avec VLC, permet la lecture automatique des épisodes d'une série.
//
// Paramètres query :
// - folder : chemin du dossier (requis)
// - token : token d'authentification (optionnel, sera ajouté aux URLs si présent)
//
// Exemple de requête : GET /api/playlist?folder=D:/Videos/Series/Season1&token=xyz
//
// Réponse (format M3U) :
//
//	#EXTM3U
//	#EXTINF:-1,Episode_01.mkv
//	http://192.168.1.10:8080/stream/abc123?token=xyz
//	#EXTINF:-1,Episode_02.mkv
//	http://192.168.1.10:8080/stream/def456?token=xyz
func (s *Server) handlePlaylist(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get("folder")
	if folder == "" {
		http.Error(w, "Paramètre 'folder' manquant", http.StatusBadRequest)
		return
	}

	token := r.URL.Query().Get("token")

	// Récupération de tous les médias du dossier
	medias := s.index.GetByFolder(folder)
	if len(medias) == 0 {
		http.Error(w, "Aucun média trouvé dans ce dossier", http.StatusNotFound)
		return
	}

	// Construction de l'URL de base pour le streaming
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	// Génération du fichier M3U
	w.Header().Set("Content-Type", "audio/x-mpegurl")
	w.Header().Set("Content-Disposition", "attachment; filename=\"playlist.m3u\"")

	fmt.Fprintln(w, "#EXTM3U")

	for _, m := range medias {
		// Ligne d'info (durée inconnue = -1)
		fmt.Fprintf(w, "#EXTINF:-1,%s\n", m.Name)

		// URL de streaming
		streamURL := fmt.Sprintf("%s/stream/%s", baseURL, m.ID)
		if token != "" {
			streamURL += "?token=" + url.QueryEscape(token)
		}
		fmt.Fprintln(w, streamURL)
	}

	s.log.Info("playlist générée",
		"dossier", folder,
		"fichiers", len(medias),
		"ip", r.RemoteAddr,
	)
}

