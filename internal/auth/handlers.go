// Package auth — handlers HTTP pour l'authentification et l'administration.
package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Handler gère les routes d'authentification et d'administration.
type Handler struct {
	store *UserStore
	log   *slog.Logger
}

// NewHandler crée un nouveau handler auth.
func NewHandler(store *UserStore, log *slog.Logger) *Handler {
	return &Handler{store: store, log: log}
}

// ─── Auth publique ─────────────────────────────────────────────────────────────

// HandleRegister traite une demande d'accès (inscription).
// L'utilisateur fournit username + password.
// L'accès reste en attente jusqu'à approbation admin.
//
// POST /api/auth/register
// Body: {"username": "alice", "password": "secret"}
// Réponse: {"request_id": "xxx", "status": "pending"}
func (h *Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Corps JSON invalide", http.StatusBadRequest)
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Password = strings.TrimSpace(req.Password)

	if req.Username == "" || req.Password == "" {
		jsonError(w, "username et password sont requis", http.StatusBadRequest)
		return
	}

	if len(req.Username) < 2 || len(req.Username) > 32 {
		jsonError(w, "username doit contenir entre 2 et 32 caractères", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 4 {
		jsonError(w, "password doit contenir au moins 4 caractères", http.StatusBadRequest)
		return
	}

	// Vérification que le username n'est pas déjà pris
	if existing := h.store.GetByUsername(req.Username); existing != nil {
		if existing.Status == StatusRevoked {
			jsonError(w, "Ce compte a été révoqué", http.StatusForbidden)
		} else {
			jsonError(w, "Ce nom d'utilisateur est déjà utilisé", http.StatusConflict)
		}
		return
	}

	// Hash du mot de passe
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.log.Error("erreur hash password", slog.String("erreur", err.Error()))
		jsonError(w, "Erreur interne", http.StatusInternalServerError)
		return
	}

	user := NewUser(req.Username, string(hash))
	if err := h.store.Add(user); err != nil {
		h.log.Error("erreur ajout utilisateur", slog.String("erreur", err.Error()))
		jsonError(w, "Erreur interne", http.StatusInternalServerError)
		return
	}

	h.log.Info("nouvelle demande d'accès",
		slog.String("username", user.Username),
		slog.String("ip", r.RemoteAddr),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"request_id": user.ID,
		"status":     string(user.Status),
		"message":    "Demande enregistrée. En attente d'approbation par l'administrateur.",
	})
}

// HandleLogin vérifie les credentials et retourne le token si approuvé,
// ou le statut si en attente.
//
// POST /api/auth/login
// Body: {"username": "alice", "password": "secret"}
// Réponse approuvé: {"token": "xxx", "status": "approved", "username": "alice"}
// Réponse pending:  {"status": "pending", "request_id": "xxx"}
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Corps JSON invalide", http.StatusBadRequest)
		return
	}

	req.Username = strings.TrimSpace(req.Username)

	user := h.store.GetByUsername(req.Username)
	if user == nil {
		// Réponse générique pour éviter l'enumeration
		jsonError(w, "Nom d'utilisateur ou mot de passe incorrect", http.StatusUnauthorized)
		return
	}

	// Vérification du mot de passe
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		jsonError(w, "Nom d'utilisateur ou mot de passe incorrect", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch user.Status {
	case StatusPending:
		json.NewEncoder(w).Encode(map[string]string{
			"status":     "pending",
			"request_id": user.ID,
			"message":    "Votre compte est en attente d'approbation par l'administrateur.",
		})

	case StatusApproved:
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "approved",
			"token":    user.Token,
			"username": user.Username,
		})

	case StatusRevoked:
		jsonError(w, "Votre accès a été révoqué par l'administrateur.", http.StatusForbidden)
	}
}

// HandleStatus permet à un utilisateur de vérifier l'état de sa demande.
//
// GET /api/auth/status?request_id=xxx
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		jsonError(w, "request_id manquant", http.StatusBadRequest)
		return
	}

	user := h.store.GetByID(requestID)
	if user == nil {
		jsonError(w, "Demande introuvable", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	resp := map[string]string{
		"status":   string(user.Status),
		"username": user.Username,
	}

	if user.Status == StatusApproved {
		resp["token"] = user.Token
	}

	json.NewEncoder(w).Encode(resp)
}

// ─── Administration (localhost uniquement) ─────────────────────────────────────

// HandleAdminListUsers retourne tous les utilisateurs.
// GET /admin/users
func (h *Handler) HandleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users := h.store.List()

	// Tri: pending en premier, puis approved, puis revoked
	sort.Slice(users, func(i, j int) bool {
		order := map[UserStatus]int{StatusPending: 0, StatusApproved: 1, StatusRevoked: 2}
		if order[users[i].Status] != order[users[j].Status] {
			return order[users[i].Status] < order[users[j].Status]
		}
		return users[i].RequestedAt.After(users[j].RequestedAt)
	})

	// On masque les mots de passe dans la réponse
	type safeUser struct {
		ID          string     `json:"id"`
		Username    string     `json:"username"`
		Status      UserStatus `json:"status"`
		RequestedAt string     `json:"requested_at"`
		ApprovedAt  *string    `json:"approved_at,omitempty"`
	}

	safe := make([]safeUser, len(users))
	for i, u := range users {
		su := safeUser{
			ID:          u.ID,
			Username:    u.Username,
			Status:      u.Status,
			RequestedAt: u.RequestedAt.Format("2006-01-02 15:04:05"),
		}
		if u.ApprovedAt != nil {
			t := u.ApprovedAt.Format("2006-01-02 15:04:05")
			su.ApprovedAt = &t
		}
		safe[i] = su
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total":   len(users),
		"pending": len(h.store.ListPending()),
		"users":   safe,
	})
}

// HandleAdminApprove approuve un utilisateur et génère son token.
// POST /admin/users/{id}/approve
func (h *Handler) HandleAdminApprove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonError(w, "ID manquant", http.StatusBadRequest)
		return
	}

	user, err := h.store.Approve(id)
	if err != nil {
		h.log.Error("erreur approbation", slog.String("erreur", err.Error()))
		jsonError(w, "Erreur interne", http.StatusInternalServerError)
		return
	}
	if user == nil {
		jsonError(w, "Utilisateur introuvable", http.StatusNotFound)
		return
	}

	h.log.Info("utilisateur approuvé par admin",
		slog.String("username", user.Username),
		slog.String("ip", r.RemoteAddr),
	)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message":  "Utilisateur approuvé",
		"username": user.Username,
	})
}

// HandleAdminRevoke révoque l'accès d'un utilisateur.
// POST /admin/users/{id}/revoke
func (h *Handler) HandleAdminRevoke(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonError(w, "ID manquant", http.StatusBadRequest)
		return
	}

	removed, err := h.store.Revoke(id)
	if err != nil {
		h.log.Error("erreur révocation", slog.String("erreur", err.Error()))
		jsonError(w, "Erreur interne", http.StatusInternalServerError)
		return
	}
	if !removed {
		jsonError(w, "Utilisateur introuvable", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Accès révoqué"})
}

// HandleAdminDelete supprime définitivement un utilisateur.
// DELETE /admin/users/{id}
func (h *Handler) HandleAdminDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		jsonError(w, "ID manquant", http.StatusBadRequest)
		return
	}

	removed, err := h.store.Revoke(id) // Revoke = désactive le token
	if err != nil {
		jsonError(w, "Erreur interne", http.StatusInternalServerError)
		return
	}
	if !removed {
		jsonError(w, "Utilisateur introuvable", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Utilisateur supprimé"})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
