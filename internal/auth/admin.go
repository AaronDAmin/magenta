// Package auth — handlers HTTP pour la gestion des appareils.
package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// AdminHandler gère les routes d'administration des appareils.
type AdminHandler struct {
	store *Store
	log   *slog.Logger
}

// NewAdminHandler crée un nouveau handler admin.
func NewAdminHandler(store *Store, log *slog.Logger) *AdminHandler {
	return &AdminHandler{
		store: store,
		log:   log,
	}
}

// HandleList retourne la liste de tous les appareils autorisés.
// Route : GET /admin/devices
//
// Réponse :
//
//	{
//	  "total": 2,
//	  "devices": [
//	    {
//	      "id": "abc123",
//	      "name": "iPhone",
//	      "token": "secret-token-1",
//	      "created_at": "2026-09-16T11:00:00Z"
//	    },
//	    ...
//	  ]
//	}
func (h *AdminHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	devices := h.store.List()

	response := map[string]interface{}{
		"total":   len(devices),
		"devices": devices,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleAdd crée un nouvel appareil et retourne ses informations (y compris le token).
// Route : POST /admin/devices
//
// Body :
//
//	{
//	  "name": "iPhone de Marie"
//	}
//
// Réponse :
//
//	{
//	  "id": "xyz789",
//	  "name": "iPhone de Marie",
//	  "token": "secret-token-xyz",
//	  "created_at": "2026-09-16T11:30:00Z"
//	}
func (h *AdminHandler) HandleAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Body JSON invalide", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Le champ 'name' est requis", http.StatusBadRequest)
		return
	}

	// Création du nouvel appareil
	device := NewDevice(req.Name)

	// Ajout au store
	if err := h.store.Add(device); err != nil {
		h.log.Error("erreur lors de l'ajout de l'appareil",
			slog.String("erreur", err.Error()),
		)
		http.Error(w, "Erreur interne du serveur", http.StatusInternalServerError)
		return
	}

	// Retourne l'appareil créé (avec le token)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(device)
}

// HandleRevoke révoque un appareil (supprime son accès).
// Route : DELETE /admin/devices/{id}
//
// Réponse si succès : 200 OK
// Réponse si appareil introuvable : 404 Not Found
func (h *AdminHandler) HandleRevoke(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "ID manquant dans l'URL", http.StatusBadRequest)
		return
	}

	removed, err := h.store.Remove(id)
	if err != nil {
		h.log.Error("erreur lors de la suppression de l'appareil",
			slog.String("id", id),
			slog.String("erreur", err.Error()),
		)
		http.Error(w, "Erreur interne du serveur", http.StatusInternalServerError)
		return
	}

	if !removed {
		http.Error(w, "Appareil introuvable", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Appareil révoqué avec succès"}`))
}
