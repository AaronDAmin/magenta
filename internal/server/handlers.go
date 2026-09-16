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
