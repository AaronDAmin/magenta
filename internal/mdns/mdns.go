// Package mdns gère la découverte réseau via mDNS (multicast DNS).
// Il permet au serveur de s'annoncer automatiquement sur le réseau local
// sous le nom "magenta.local" sans configuration manuelle sur les clients.
//
// Protocole : mDNS / DNS-SD (RFC 6762 / RFC 6763)
// Bibliothèque : github.com/grandcat/zeroconf
//
// Résultat : tous les appareils du réseau peuvent accéder au serveur
// via http://magenta.local:8080 sans connaître l'adresse IP.
package mdns

import (
	"fmt"
	"log/slog"

	"github.com/grandcat/zeroconf"
)

// Server encapsule le serveur mDNS.
type Server struct {
	log      *slog.Logger
	instance *zeroconf.Server // serveur zeroconf sous-jacent
}

// New crée et démarre un serveur mDNS qui annonce le service Magenta.
//
// Paramètres :
//   - instanceName : nom de l'instance affiché sur le réseau (ex: "magenta")
//   - port : port TCP du serveur HTTP (ex: 8080)
//
// Le serveur sera accessible via :
//   - http://magenta.local:8080  (sur macOS, Linux, iOS, Android)
//   - http://magenta.local:8080  (sur Windows avec Bonjour/mDNS activé)
func New(log *slog.Logger, instanceName string, port int) (*Server, error) {
	// Enregistrement du service mDNS avec le type _http._tcp
	// Ce type standard permet à tous les clients mDNS de découvrir le service.
	instance, err := zeroconf.Register(
		instanceName,  // Nom de l'instance (ex: "magenta")
		"_http._tcp",  // Type de service standard pour HTTP
		"local.",      // Domaine (toujours "local." pour mDNS)
		port,          // Port du serveur HTTP
		[]string{
			// Métadonnées TXT optionnelles — utiles pour les clients
			"version=1",
			fmt.Sprintf("path=/"),
		},
		nil, // Utilise toutes les interfaces réseau disponibles
	)
	if err != nil {
		return nil, fmt.Errorf("enregistrement mDNS : %w", err)
	}

	log.Info("mDNS actif",
		slog.String("nom", instanceName+".local"),
		slog.Int("port", port),
		slog.String("url", fmt.Sprintf("http://%s.local:%d", instanceName, port)),
	)

	return &Server{
		log:      log,
		instance: instance,
	}, nil
}

// Stop arrête l'annonce mDNS proprement.
// Les autres appareils du réseau cesseront de voir le service.
func (s *Server) Stop() {
	if s.instance != nil {
		s.instance.Shutdown()
		s.log.Info("mDNS arrêté")
	}
}
