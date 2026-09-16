# Magenta 🎬

Serveur multimédia local privé développé en Go.  
Accédez à vos films, séries et vidéos depuis n'importe quel appareil connecté à votre réseau local — sans cloud, sans abonnement, sans Internet.

## Fonctionnement

- Le serveur tourne sur l'ordinateur qui contient vos fichiers
- Les appareils autorisés accèdent aux contenus via HTTP
- La lecture se fait directement depuis VLC ou tout lecteur compatible
- Le système fonctionne entièrement en réseau local

## Prérequis

- Go 1.21+
- VLC (ou tout lecteur supportant HTTP avec range requests)

## Démarrage rapide

```bash
# Cloner le repo
git clone https://github.com/AaronDAmin/magenta.git
cd magenta

# Éditer la configuration
# Modifier config.yaml : renseigner vos dossiers médias et votre IP locale

# Compiler
go build -o bin/magenta ./cmd/mediaserver

# Lancer
./bin/magenta -config config.yaml
```

## Configuration

Modifier `config.yaml` à la racine du projet :

```yaml
server:
  host: "192.168.1.10"  # Votre IP locale
  port: 8080

media:
  dirs:
    - "D:/Videos"
    - "D:/Films"
```

## Architecture

```
magenta/
├── cmd/mediaserver/     # Point d'entrée
├── internal/
│   ├── config/          # Chargement de la configuration
│   ├── logger/          # Logger structuré
│   ├── server/          # Serveur HTTP, routes, middleware
│   ├── media/           # Scan et index des médias
│   ├── stream/          # Streaming HTTP avec range requests
│   └── auth/            # Authentification par token
└── web/                 # Interface web embarquée
```

## Roadmap

- [x] Étape 1 — Fondations (config, logger, structure)
- [ ] Étape 2 — Serveur HTTP de base
- [ ] Étape 3 — Scan et index des médias
- [ ] Étape 4 — Streaming des fichiers
- [ ] Étape 5 — Authentification par token
- [ ] Étape 6 — Interface web embarquée
- [ ] Étape 7 — Découverte réseau mDNS
- [ ] Étape 8 — TLS local (HTTPS)

## Licence

Projet privé — usage personnel uniquement.
