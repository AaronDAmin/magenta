// Package server — handlers HTTP.
// Ce fichier contient tous les gestionnaires de routes HTTP du serveur.
package server

import (
	"encoding/json"
	"net/http"
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

