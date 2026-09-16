// Package auth — stockage et gestion des appareils autorisés.
package auth

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
)

// Store gère la persistance des appareils autorisés dans un fichier JSON.
// Il est thread-safe et peut être utilisé depuis plusieurs goroutines.
type Store struct {
	filePath string
	log      *slog.Logger
	mu       sync.RWMutex
	devices  map[string]*Device // clé = token, valeur = Device
}

// storeData est la structure du fichier JSON.
type storeData struct {
	Devices []*Device `json:"devices"`
}

// NewStore crée un nouveau store et charge les appareils depuis le fichier.
// Si le fichier n'existe pas, il sera créé au premier ajout d'appareil.
func NewStore(filePath string, log *slog.Logger) (*Store, error) {
	s := &Store{
		filePath: filePath,
		log:      log,
		devices:  make(map[string]*Device),
	}

	// Tentative de chargement du fichier existant
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("chargement du store : %w", err)
	}

	return s, nil
}

// load charge les appareils depuis le fichier JSON.
func (s *Store) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var sd storeData
	if err := json.Unmarshal(data, &sd); err != nil {
		return fmt.Errorf("parsing JSON : %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Reconstruction de la map depuis la liste
	s.devices = make(map[string]*Device)
	for _, d := range sd.Devices {
		s.devices[d.Token] = d
	}

	s.log.Info("appareils chargés", slog.Int("total", len(s.devices)))
	return nil
}

// save sauvegarde les appareils dans le fichier JSON.
func (s *Store) save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Conversion de la map en liste pour le JSON
	devices := make([]*Device, 0, len(s.devices))
	for _, d := range s.devices {
		devices = append(devices, d)
	}

	sd := storeData{Devices: devices}

	data, err := json.MarshalIndent(sd, "", "  ")
	if err != nil {
		return fmt.Errorf("encodage JSON : %w", err)
	}

	// Écriture atomique : écriture dans un fichier temporaire puis rename
	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("écriture du fichier temporaire : %w", err)
	}

	if err := os.Rename(tmpPath, s.filePath); err != nil {
		return fmt.Errorf("rename du fichier : %w", err)
	}

	return nil
}

// Add ajoute un nouvel appareil et sauvegarde immédiatement.
func (s *Store) Add(device *Device) error {
	s.mu.Lock()
	s.devices[device.Token] = device
	s.mu.Unlock()

	if err := s.save(); err != nil {
		return fmt.Errorf("sauvegarde après ajout : %w", err)
	}

	s.log.Info("appareil ajouté",
		slog.String("id", device.ID),
		slog.String("nom", device.Name),
	)

	return nil
}

// Remove supprime un appareil par son ID et sauvegarde immédiatement.
// Retourne true si l'appareil existait et a été supprimé.
func (s *Store) Remove(id string) (bool, error) {
	s.mu.Lock()

	// Recherche de l'appareil par ID
	var found bool
	var tokenToRemove string
	for token, device := range s.devices {
		if device.ID == id {
			tokenToRemove = token
			found = true
			break
		}
	}

	if !found {
		s.mu.Unlock()
		return false, nil
	}

	delete(s.devices, tokenToRemove)
	s.mu.Unlock()

	if err := s.save(); err != nil {
		return true, fmt.Errorf("sauvegarde après suppression : %w", err)
	}

	s.log.Info("appareil supprimé", slog.String("id", id))

	return true, nil
}

// GetByToken récupère un appareil par son token.
// Retourne nil si le token n'existe pas.
func (s *Store) GetByToken(token string) *Device {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.devices[token]
}

// List retourne la liste de tous les appareils autorisés.
func (s *Store) List() []*Device {
	s.mu.RLock()
	defer s.mu.RUnlock()

	devices := make([]*Device, 0, len(s.devices))
	for _, d := range s.devices {
		devices = append(devices, d)
	}
	return devices
}

// Count retourne le nombre d'appareils autorisés.
func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.devices)
}
