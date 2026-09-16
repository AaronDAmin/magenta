// Package server — handlers HTTP.
// Ce fichier contient tous les gestionnaires de routes HTTP du serveur.
package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AaronDAmin/magenta/internal/media"
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
func (s *Server) handleMediaList(w http.ResponseWriter, r *http.Request) {
	allMedia := s.index.All()
	response := map[string]interface{}{
		"total": len(allMedia),
		"items": allMedia,
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error("erreur encodage /api/media", "erreur", err.Error())
	}
}

// handleFolders retourne les médias groupés par dossier.
// Chaque groupe représente soit un film isolé, soit une série avec ses épisodes.
//
// Réponse :
//
//	{
//	  "groups": [
//	    {
//	      "name": "THAT TIME I GOT REINCARNATED",
//	      "folder": "D:/Series/THAT TIME...",
//	      "count": 12,
//	      "items": [...]
//	    },
//	    ...
//	  ]
//	}
func (s *Server) handleFolders(w http.ResponseWriter, r *http.Request) {
	allMedia := s.index.All()

	// Regroupement par dossier
	folderMap := make(map[string][]*media.Media)
	for _, m := range allMedia {
		folderMap[m.Folder] = append(folderMap[m.Folder], m)
	}

	// Construction des groupes triés par nom
	type Group struct {
		Name   string        `json:"name"`
		Folder string        `json:"folder"`
		Count  int           `json:"count"`
		Items  []*media.Media `json:"items"`
	}

	groups := make([]Group, 0, len(folderMap))
	for folder, items := range folderMap {
		// Tri des épisodes par nom dans chaque groupe
		sortByName(items)

		// Nom du groupe = dernier segment du chemin (nom du dossier)
		name := folderName(folder)

		groups = append(groups, Group{
			Name:   name,
			Folder: folder,
			Count:  len(items),
			Items:  items,
		})
	}

	// Tri des groupes par nom
	sort.Slice(groups, func(i, j int) bool {
		return strings.ToLower(groups[i].Name) < strings.ToLower(groups[j].Name)
	})

	response := map[string]interface{}{
		"total":  len(allMedia),
		"groups": groups,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log.Error("erreur encodage /api/folders", "erreur", err.Error())
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

// handleVLC génère un fichier .m3u à un seul élément pour ouvrir directement dans VLC.
// Le navigateur télécharge le fichier, VLC l'ouvre automatiquement (association de fichier).
// Route : GET /stream/{id}/vlc?token=xyz
func (s *Server) handleVLC(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "ID manquant", http.StatusBadRequest)
		return
	}

	m := s.index.Get(id)
	if m == nil {
		http.Error(w, "Média introuvable", http.StatusNotFound)
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		// Essai depuis le cookie
		if cookie, err := r.Cookie("magenta_token"); err == nil {
			token = cookie.Value
		}
	}

	// Construction de l'URL de streaming
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	streamURL := fmt.Sprintf("%s://%s/stream/%s", scheme, r.Host, id)
	if token != "" {
		streamURL += "?token=" + url.QueryEscape(token)
	}

	// Nom du fichier .m3u basé sur le nom du média
	filename := strings.TrimSuffix(m.Name, "."+m.Extension) + ".m3u"

	// Headers pour déclencher le téléchargement
	w.Header().Set("Content-Type", "audio/x-mpegurl")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	// Contenu M3U minimal — VLC l'ouvre directement
	fmt.Fprintln(w, "#EXTM3U")
	fmt.Fprintf(w, "#EXTINF:-1,%s\n", m.Name)
	fmt.Fprintln(w, streamURL)

	s.log.Info("fichier VLC généré",
		slog.String("id", id),
		slog.String("nom", m.Name),
		slog.String("ip", r.RemoteAddr),
	)
}


// ─── Helpers ──────────────────────────────────────────────────────────────────

// folderName retourne le dernier segment d'un chemin (nom du dossier).
func folderName(path string) string {
	name := filepath.Base(filepath.Clean(path))
	// Nettoyage : suppression des underscores, mise en forme lisible
	name = strings.ReplaceAll(name, "_", " ")
	return name
}

// sortByName trie une liste de médias par nom alphabétique.
func sortByName(items []*media.Media) {
	sort.Slice(items, func(i, j int) bool {
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
}
