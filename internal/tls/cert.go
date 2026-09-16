// Package tls gère la génération et le chargement des certificats TLS auto-signés.
// Le certificat est généré automatiquement au premier démarrage et réutilisé ensuite.
// Cela permet d'activer HTTPS sur le réseau local sans CA publique.
package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"os"
	"time"
)

// CertConfig contient les chemins des fichiers de certificat et clé.
type CertConfig struct {
	CertFile string // Chemin du fichier certificat (.crt ou .pem)
	KeyFile  string // Chemin du fichier clé privée (.key)
}

// LoadOrGenerate charge un certificat TLS existant ou en génère un nouveau.
// Si les fichiers n'existent pas, un certificat auto-signé est généré automatiquement.
// Le certificat est valide pour 10 ans et couvre les noms/IPs configurés.
func LoadOrGenerate(cfg CertConfig, log *slog.Logger, hosts []string) (*tls.Config, error) {
	// Vérification si les fichiers existent déjà
	if fileExists(cfg.CertFile) && fileExists(cfg.KeyFile) {
		log.Info("chargement du certificat TLS existant",
			slog.String("cert", cfg.CertFile),
		)
		return loadTLSConfig(cfg.CertFile, cfg.KeyFile)
	}

	// Génération d'un nouveau certificat auto-signé
	log.Info("génération d'un nouveau certificat TLS auto-signé",
		slog.String("cert", cfg.CertFile),
		slog.String("key", cfg.KeyFile),
	)

	if err := generateSelfSigned(cfg.CertFile, cfg.KeyFile, hosts); err != nil {
		return nil, fmt.Errorf("génération du certificat : %w", err)
	}

	log.Info("certificat TLS généré avec succès",
		slog.String("cert", cfg.CertFile),
		slog.Duration("validité", 10*365*24*time.Hour),
	)

	return loadTLSConfig(cfg.CertFile, cfg.KeyFile)
}

// generateSelfSigned génère un certificat auto-signé avec une clé ECDSA P-256.
// ECDSA est préféré à RSA pour ses meilleures performances sur les petits appareils.
func generateSelfSigned(certFile, keyFile string, hosts []string) error {
	// Génération de la clé privée ECDSA P-256
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("génération de la clé privée : %w", err)
	}

	// Numéro de série aléatoire (requis par X.509)
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("génération du numéro de série : %w", err)
	}

	now := time.Now()

	// Template du certificat X.509
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Magenta Media Server"},
			CommonName:   "magenta.local",
		},
		NotBefore: now,
		NotAfter:  now.Add(10 * 365 * 24 * time.Hour), // valide 10 ans

		// Usages du certificat
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Ajout des noms d'hôtes et IPs au certificat (SAN — Subject Alternative Names)
	for _, host := range hosts {
		if ip := net.ParseIP(host); ip != nil {
			// C'est une adresse IP
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			// C'est un nom de domaine
			template.DNSNames = append(template.DNSNames, host)
		}
	}

	// Toujours inclure localhost et magenta.local
	template.DNSNames = append(template.DNSNames, "localhost", "magenta.local")
	template.IPAddresses = append(template.IPAddresses,
		net.ParseIP("127.0.0.1"),
		net.ParseIP("::1"),
	)

	// Génération du certificat auto-signé (signé par sa propre clé)
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("création du certificat : %w", err)
	}

	// Écriture du certificat en PEM
	certOut, err := os.OpenFile(certFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("ouverture du fichier certificat : %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return fmt.Errorf("encodage du certificat PEM : %w", err)
	}

	// Écriture de la clé privée en PEM (permissions restreintes)
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("ouverture du fichier clé : %w", err)
	}
	defer keyOut.Close()

	privDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("encodage de la clé privée : %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER}); err != nil {
		return fmt.Errorf("encodage de la clé PEM : %w", err)
	}

	return nil
}

// loadTLSConfig charge un certificat TLS depuis les fichiers PEM.
func loadTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("chargement du certificat : %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		// Paramètres TLS modernes et sécurisés
		MinVersion: tls.VersionTLS12,
		// Suites de chiffrement performantes et sécurisées
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
		},
	}, nil
}

// fileExists retourne true si le fichier existe.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
