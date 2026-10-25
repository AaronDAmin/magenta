// Package auth — modèle utilisateur avec système d'approbation.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// UserStatus représente l'état d'un utilisateur dans le système.
type UserStatus string

const (
	// StatusPending : l'utilisateur a demandé l'accès, en attente d'approbation admin.
	StatusPending UserStatus = "pending"

	// StatusApproved : l'utilisateur est approuvé, il possède un token valide.
	StatusApproved UserStatus = "approved"

	// StatusRevoked : l'accès de l'utilisateur a été révoqué par l'admin.
	StatusRevoked UserStatus = "revoked"
)

// User représente un utilisateur du système Magenta.
type User struct {
	// ID est un identifiant unique généré à la création.
	ID string `json:"id"`

	// Username est le nom d'utilisateur choisi lors de l'inscription.
	Username string `json:"username"`

	// PasswordHash est le hash bcrypt du mot de passe.
	PasswordHash string `json:"password_hash"`

	// Token est le token d'accès généré après approbation.
	// Vide si l'utilisateur est en attente ou révoqué.
	Token string `json:"token,omitempty"`

	// Status est l'état actuel de l'utilisateur.
	Status UserStatus `json:"status"`

	// RequestedAt est la date de la demande d'accès.
	RequestedAt time.Time `json:"requested_at"`

	// ApprovedAt est la date d'approbation (nil si pas encore approuvé).
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
}

// NewUser crée un nouvel utilisateur en état "pending".
func NewUser(username, passwordHash string) *User {
	return &User{
		ID:           generateID(),
		Username:     username,
		PasswordHash: passwordHash,
		Status:       StatusPending,
		RequestedAt:  time.Now(),
	}
}

// Approve approuve l'utilisateur et génère un token d'accès.
func (u *User) Approve() {
	u.Status = StatusApproved
	u.Token = generateToken()
	now := time.Now()
	u.ApprovedAt = &now
}

// Revoke révoque l'accès de l'utilisateur.
func (u *User) Revoke() {
	u.Status = StatusRevoked
	u.Token = ""
}

// IsApproved retourne true si l'utilisateur est approuvé et possède un token valide.
func (u *User) IsApproved() bool {
	return u.Status == StatusApproved && u.Token != ""
}

// generateID génère un identifiant unique (16 octets = 32 caractères hex).
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateToken génère un token d'authentification sécurisé (32 octets = 64 caractères hex).
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
