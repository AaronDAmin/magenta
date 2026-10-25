// Package auth — stockage et gestion des utilisateurs.
package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

// UserStore gère la persistance des utilisateurs dans un fichier JSON.
// Thread-safe, peut être utilisé depuis plusieurs goroutines simultanément.
type UserStore struct {
	filePath string
	log      *slog.Logger
	mu       sync.RWMutex
	byID     map[string]*User // clé = ID utilisateur
	byToken  map[string]*User // clé = token (uniquement les approuvés)
	byName   map[string]*User // clé = username (lowercase)
}

type userStoreData struct {
	Users []*User `json:"users"`
}

// NewUserStore crée un store et charge les utilisateurs depuis le fichier.
// Si le fichier n'existe pas, il sera créé au premier ajout.
func NewUserStore(filePath string, log *slog.Logger) (*UserStore, error) {
	s := &UserStore{
		filePath: filePath,
		log:      log,
		byID:     make(map[string]*User),
		byToken:  make(map[string]*User),
		byName:   make(map[string]*User),
	}

	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("chargement du store utilisateurs : %w", err)
	}

	return s, nil
}

// load charge les utilisateurs depuis le fichier JSON.
func (s *UserStore) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var sd userStoreData
	if err := json.Unmarshal(data, &sd); err != nil {
		return fmt.Errorf("parsing JSON : %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.byID = make(map[string]*User)
	s.byToken = make(map[string]*User)
	s.byName = make(map[string]*User)

	for _, u := range sd.Users {
		s.byID[u.ID] = u
		s.byName[u.Username] = u
		if u.IsApproved() {
			s.byToken[u.Token] = u
		}
	}

	s.log.Info("utilisateurs chargés", slog.Int("total", len(s.byID)))
	return nil
}

// save sauvegarde les utilisateurs dans le fichier JSON (écriture atomique).
func (s *UserStore) save() error {
	s.mu.RLock()
	users := make([]*User, 0, len(s.byID))
	for _, u := range s.byID {
		users = append(users, u)
	}
	s.mu.RUnlock()

	sd := userStoreData{Users: users}
	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return fmt.Errorf("encodage JSON : %w", err)
	}

	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("écriture temporaire : %w", err)
	}
	if err := os.Rename(tmpPath, s.filePath); err != nil {
		return fmt.Errorf("rename : %w", err)
	}

	return nil
}

// Add ajoute un nouvel utilisateur (état pending).
func (s *UserStore) Add(u *User) error {
	s.mu.Lock()
	s.byID[u.ID] = u
	s.byName[u.Username] = u
	s.mu.Unlock()

	if err := s.save(); err != nil {
		return fmt.Errorf("sauvegarde après ajout : %w", err)
	}

	s.log.Info("nouvel utilisateur enregistré",
		slog.String("id", u.ID),
		slog.String("username", u.Username),
	)
	return nil
}

// Approve approuve un utilisateur par son ID et génère son token.
// Retourne l'utilisateur mis à jour ou nil si introuvable.
func (s *UserStore) Approve(id string) (*User, error) {
	s.mu.Lock()
	u, ok := s.byID[id]
	if !ok {
		s.mu.Unlock()
		return nil, nil
	}
	u.Approve()
	s.byToken[u.Token] = u
	s.mu.Unlock()

	if err := s.save(); err != nil {
		return nil, fmt.Errorf("sauvegarde après approbation : %w", err)
	}

	s.log.Info("utilisateur approuvé",
		slog.String("id", u.ID),
		slog.String("username", u.Username),
	)
	return u, nil
}

// Revoke révoque l'accès d'un utilisateur par son ID.
func (s *UserStore) Revoke(id string) (bool, error) {
	s.mu.Lock()
	u, ok := s.byID[id]
	if !ok {
		s.mu.Unlock()
		return false, nil
	}
	oldToken := u.Token
	u.Revoke()
	delete(s.byToken, oldToken)
	s.mu.Unlock()

	if err := s.save(); err != nil {
		return true, fmt.Errorf("sauvegarde après révocation : %w", err)
	}

	s.log.Info("utilisateur révoqué",
		slog.String("id", u.ID),
		slog.String("username", u.Username),
	)
	return true, nil
}

// GetByToken récupère un utilisateur approuvé par son token.
func (s *UserStore) GetByToken(token string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byToken[token]
}

// GetByUsername récupère un utilisateur par son nom (insensible à la casse).
func (s *UserStore) GetByUsername(username string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byName[username]
}

// GetByID récupère un utilisateur par son ID.
func (s *UserStore) GetByID(id string) *User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.byID[id]
}

// List retourne tous les utilisateurs.
func (s *UserStore) List() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]*User, 0, len(s.byID))
	for _, u := range s.byID {
		users = append(users, u)
	}
	return users
}

// ListPending retourne les utilisateurs en attente d'approbation.
func (s *UserStore) ListPending() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var pending []*User
	for _, u := range s.byID {
		if u.Status == StatusPending {
			pending = append(pending, u)
		}
	}
	return pending
}

// Count retourne le nombre total d'utilisateurs.
func (s *UserStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.byID)
}
