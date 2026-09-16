// Package auth gère l'authentification des appareils via des tokens.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Device représente un appareil autorisé à accéder au serveur.
type Device struct {
	// ID est un identifiant unique de l'appareil (UUID).
	ID string `json:"id"`

	// Name est le nom donné à l'appareil par l'utilisateur (ex: "iPhone de Marie").
	Name string `json:"name"`

	// Token est la clé secrète utilisée pour l'authentification.
	// Ce token doit être gardé confidentiel par l'appareil.
	Token string `json:"token"`

	// CreatedAt est la date de création de l'appareil.
	CreatedAt time.Time `json:"created_at"`
}

// NewDevice crée un nouvel appareil avec un ID et un token générés automatiquement.
func NewDevice(name string) *Device {
	return &Device{
		ID:        generateID(),
		Name:      name,
		Token:     generateToken(),
		CreatedAt: time.Now(),
	}
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
