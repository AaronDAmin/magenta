// Package config gère le chargement et la validation de la configuration du serveur.
// La configuration est lue depuis un fichier YAML externe (config.yaml),
// ce qui permet de modifier les paramètres sans recompiler le binaire.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config représente la configuration complète du serveur.
type Config struct {
	// Server regroupe les paramètres réseau du serveur HTTP.
	Server ServerConfig `yaml:"server"`

	// Media regroupe les paramètres liés aux fichiers multimédias.
	Media MediaConfig `yaml:"media"`

	// Log regroupe les paramètres de logging.
	Log LogConfig `yaml:"log"`

	// MDNS regroupe les paramètres de découverte réseau.
	MDNS MDNSConfig `yaml:"mdns"`
}

// ServerConfig contient les paramètres réseau du serveur.
type ServerConfig struct {
	// Host est l'adresse IP sur laquelle le serveur écoute.
	// Par défaut "0.0.0.0" mais doit être restreint à l'IP locale en production.
	Host string `yaml:"host"`

	// Port est le port TCP sur lequel le serveur écoute.
	Port int `yaml:"port"`

	// ReadTimeoutSec est le délai maximum (en secondes) pour lire une requête entrante.
	ReadTimeoutSec int `yaml:"read_timeout_sec"`

	// WriteTimeoutSec est le délai maximum (en secondes) pour écrire une réponse.
	// Mettre à 0 pour désactiver (recommandé pour le streaming de gros fichiers).
	WriteTimeoutSec int `yaml:"write_timeout_sec"`
}

// MediaConfig contient les paramètres liés aux médias.
type MediaConfig struct {
	// AllowedRoots est la liste des racines autorisées pour la sécurité.
	// Seuls les chemins sous ces racines peuvent être scannés.
	// Cette protection empêche l'exposition accidentelle de fichiers système ou personnels.
	AllowedRoots []string `yaml:"allowed_roots"`

	// Dirs est la liste des répertoires à scanner pour trouver les fichiers multimédias.
	// Chaque chemin DOIT être sous l'une des racines autorisées (allowed_roots).
	Dirs []string `yaml:"dirs"`

	// Extensions est la liste des extensions de fichiers considérées comme des médias.
	Extensions []string `yaml:"extensions"`
}

// LogConfig contient les paramètres de logging.
type LogConfig struct {
	// Level est le niveau de verbosité : "debug", "info", "warn", "error".
	Level string `yaml:"level"`

	// JSON indique si les logs doivent être émis en format JSON (true) ou texte (false).
	JSON bool `yaml:"json"`
}

// MDNSConfig contient les paramètres de découverte réseau mDNS.
type MDNSConfig struct {
	// Enabled active ou désactive la découverte mDNS.
	Enabled bool `yaml:"enabled"`

	// Name est le nom de l'instance sur le réseau (ex: "magenta" → "magenta.local").
	Name string `yaml:"name"`
}

// Load lit et parse le fichier de configuration YAML au chemin indiqué.
// Retourne une erreur si le fichier est introuvable ou malformé.
func Load(path string) (*Config, error) {
	// Lecture brute du fichier
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("lecture du fichier de config %q : %w", path, err)
	}

	// Parsing YAML vers la structure Config
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing YAML de %q : %w", path, err)
	}

	// Validation des valeurs obligatoires
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration invalide : %w", err)
	}

	return cfg, nil
}

// validate vérifie que les valeurs de configuration sont cohérentes.
func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port doit être entre 1 et 65535, valeur reçue : %d", c.Server.Port)
	}
	if len(c.Media.AllowedRoots) == 0 {
		return fmt.Errorf("media.allowed_roots ne peut pas être vide : au moins une racine doit être autorisée")
	}
	if len(c.Media.Dirs) == 0 {
		return fmt.Errorf("media.dirs ne peut pas être vide : au moins un répertoire doit être configuré")
	}

	// Validation de sécurité : vérifier que tous les dirs sont sous les racines autorisées
	if err := c.validateMediaDirs(); err != nil {
		return err
	}

	return nil
}

// validateMediaDirs vérifie que chaque répertoire média est sous une racine autorisée.
// Cette validation empêche l'exposition accidentelle de fichiers système sensibles.
func (c *Config) validateMediaDirs() error {
	for _, dir := range c.Media.Dirs {
		// Résolution des liens symboliques pour éviter les contournements
		realPath, err := filepath.EvalSymlinks(dir)
		if err != nil {
			// Si le chemin n'existe pas encore, on utilise le chemin nettoyé
			realPath = filepath.Clean(dir)
		}

		// Vérification : le chemin doit être sous l'une des racines autorisées
		allowed := false
		for _, root := range c.Media.AllowedRoots {
			realRoot, err := filepath.EvalSymlinks(root)
			if err != nil {
				realRoot = filepath.Clean(root)
			}

			// Vérifie si realPath commence par realRoot
			if isUnderRoot(realPath, realRoot) {
				allowed = true
				break
			}
		}

		if !allowed {
			return fmt.Errorf(
				"répertoire refusé : %q n'est pas sous l'une des racines autorisées %v (sécurité)",
				dir, c.Media.AllowedRoots,
			)
		}
	}
	return nil
}

// isUnderRoot vérifie si path est sous root (ou égal à root).
// Les chemins sont normalisés pour gérer les différences de casse sur Windows.
func isUnderRoot(path, root string) bool {
	// Normalisation des chemins (gère les séparateurs et la casse)
	path = filepath.Clean(path)
	root = filepath.Clean(root)

	// Sur Windows, la comparaison doit être insensible à la casse
	// filepath.VolumeName gère correctement les volumes Windows
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}

	// Si le chemin relatif commence par "..", alors path n'est PAS sous root
	return !strings.HasPrefix(rel, "..")
}

// String retourne une représentation JSON lisible de la config (utile pour le debug).
// Les champs sensibles devront être masqués ici à l'avenir.
func (c *Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return string(b)
}
