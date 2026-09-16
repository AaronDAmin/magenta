// Package config gère le chargement et la validation de la configuration du serveur.
// La configuration est lue depuis un fichier YAML externe (config.yaml),
// ce qui permet de modifier les paramètres sans recompiler le binaire.
package config

import (
	"encoding/json"
	"fmt"
	"os"

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
	// Dirs est la liste des répertoires à scanner pour trouver les fichiers multimédias.
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
	if len(c.Media.Dirs) == 0 {
		return fmt.Errorf("media.dirs ne peut pas être vide : au moins un répertoire doit être configuré")
	}
	return nil
}

// String retourne une représentation JSON lisible de la config (utile pour le debug).
// Les champs sensibles devront être masqués ici à l'avenir.
func (c *Config) String() string {
	b, _ := json.MarshalIndent(c, "", "  ")
	return string(b)
}
