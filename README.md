<div align="center">

<br/>

<img src="https://readme-typing-svg.demolab.com?font=Segoe+UI&weight=800&size=42&pause=1000&color=E50914&center=true&vCenter=true&width=500&lines=MAGENTA" alt="Magenta" />

<h3>✦ &nbsp; Your private Netflix — local network video streaming, no internet required &nbsp; ✦</h3>

<br/>

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey?style=for-the-badge&logo=github&logoColor=white)](https://github.com)
[![VLC](https://img.shields.io/badge/VLC-Compatible-FF8800?style=for-the-badge&logo=vlcmediaplayer&logoColor=white)](https://www.videolan.org/vlc/)
[![mDNS](https://img.shields.io/badge/mDNS-magenta.local-E50914?style=for-the-badge&logoColor=white)](https://en.wikipedia.org/wiki/Multicast_DNS)
[![License](https://img.shields.io/badge/License-GPL%20v3-E50914?style=for-the-badge)](LICENSE)

<br/>

> **Stream. Anywhere on your network. No cloud. No subscription. No tracking.**
>
> Magenta runs on your PC or homelab and exposes a Netflix-style web interface accessible
> from any device on your local network — phone, tablet, TV box, or laptop.
> Powered by a single Go binary. Zero dependencies at runtime.

<br/>

</div>

---

<div align="center">

## 📸 &nbsp; Interface Preview

<br/>

<sub><sup>✦ &nbsp; Netflix-style library · Built-in video player · Admin panel · VLC integration &nbsp; ✦</sup></sub>

<br/>

</div>

---

## ✨ &nbsp; Features

<br/>

<table>
<tr>
<td width="50%" valign="top">

### 🎬 &nbsp; Netflix-Style Interface

A dark, cinematic browsing experience for your media library.

- **Card-based library** — poster grid with folder grouping
- **Built-in video player** — seek, scrub, fullscreen
- **Series support** — episodes auto-linked, autoplay next
- **Keyboard shortcuts** — Space, ←/→, F, M
- **Search** — filter your entire library instantly

</td>
<td width="50%" valign="top">

### 🔐 &nbsp; Access Control

Simple approval-based auth — you decide who gets in.

- Register with username + password
- Admin approves or rejects requests
- Token-based sessions (cookie or query param)
- Revoke access at any time
- Admin panel accessible from localhost only

</td>
</tr>
<tr>
<td width="50%" valign="top">

### 📡 &nbsp; Streaming & Compatibility

Works with any browser or media player on your network.

- **HTTP range requests** — seek without buffering the whole file
- **VLC integration** — one-click `.m3u` launch on PC/Mac/Android
- **M3U playlists** — stream entire series folders in VLC
- **mDNS discovery** — access via `http://magenta.local:8080`
- **TLS/HTTPS** — optional self-signed certificate

</td>
<td width="50%" valign="top">

### ⚙️ &nbsp; Smart Media Management

Set it and forget it — Magenta handles the rest.

- **Auto-scan** on startup across multiple directories
- **Live file watching** — new files appear instantly (fsnotify)
- **Multi-root security** — only whitelisted paths are exposed
- **Format support** — MP4, MKV, AVI, MOV, WebM, and more
- **Single binary** — no runtime dependencies, embeds all assets

</td>
</tr>
</table>

---

## 🚀 &nbsp; Quick Start

### Prerequisites

| Requirement | Version | Link |
|---|---|---|
| Go | 1.21 or later | [go.dev](https://go.dev/dl/) |
| Video files | Any format | MP4, MKV, AVI, MOV, WebM... |

<br/>

### Installation

**1. Clone the repository**

```bash
git clone https://github.com/AaronDAmin/magenta.git
cd magenta
```

**2. Copy and edit the configuration**

```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml` to point to your media directories:

```yaml
media:
  allowed_roots:
    - "/media"          # Linux/macOS
    - "D:/"             # Windows
  dirs:
    - "/media/movies"
    - "/media/series"
```

**3. Build**

```bash
# Windows
go build -o bin/magenta.exe ./cmd/mediaserver

# Linux / macOS
go build -o bin/magenta ./cmd/mediaserver
```

**4. Run** 🎉

```bash
./bin/magenta.exe          # Windows
./bin/magenta              # Linux / macOS

# Or without building:
go run ./cmd/mediaserver
```

Open `http://localhost:8080/register` to create your first account, then approve it at `http://localhost:8080/admin`.

---

## 🔄 &nbsp; How It Works

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│  1. REGISTER  ──▶  2. ADMIN APPROVES  ──▶  3. LOGIN  ──▶  4. STREAM│
│                                                                     │
│  User requests       Admin reviews          Token issued    Browse  │
│  access at           at localhost           and stored      library │
│  /register           /admin                 in cookie       & play  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

**Step 1 — Register** &nbsp; Any device on the network visits `/register` and submits a username + password.

**Step 2 — Admin Approves** &nbsp; The admin opens `http://localhost:8080/admin` (localhost only) and approves the request.

**Step 3 — Login** &nbsp; The user logs in — a secure token is issued and stored in a cookie.

**Step 4 — Stream** &nbsp; Full access to the library. Play in-browser or open in VLC with one click.

---

## 🏗 &nbsp; Project Architecture

```
magenta/
│
├── cmd/
│   └── mediaserver/
│       ├── main.go             ← Entry point, wiring, graceful shutdown
│       └── web/                ← Embedded HTML/CSS assets (go:embed)
│
├── internal/
│   ├── auth/
│   │   ├── user.go             ← User model, token generation
│   │   ├── userstore.go        ← Thread-safe JSON persistence
│   │   ├── handlers.go         ← Register, login, status endpoints
│   │   └── middleware.go       ← Token validation middleware
│   │
│   ├── media/
│   │   ├── media.go            ← Media item model
│   │   ├── index.go            ← In-memory media index
│   │   ├── scanner.go          ← Directory scanner
│   │   └── watcher.go          ← Live file system watcher (fsnotify)
│   │
│   ├── server/
│   │   ├── server.go           ← HTTP server setup & routing
│   │   ├── handlers.go         ← Media & admin API handlers
│   │   ├── middleware.go       ← Logging, CORS, localhost guard
│   │   └── web.go              ← HTML page serving
│   │
│   ├── stream/
│   │   └── handler.go          ← HTTP range streaming handler
│   │
│   ├── config/
│   │   └── config.go           ← YAML config loader
│   │
│   ├── logger/
│   │   └── logger.go           ← slog-based structured logger
│   │
│   ├── mdns/
│   │   └── mdns.go             ← mDNS service registration
│   │
│   ├── network/
│   │   └── info.go             ← Local IP detection, startup banner
│   │
│   └── tls/
│       └── cert.go             ← TLS cert load or self-signed generation
│
├── web/                        ← Source HTML templates & CSS
│   ├── library.html
│   ├── player.html
│   ├── login.html
│   ├── register.html
│   ├── admin.html
│   └── static/
│       └── style.css
│
├── config.yaml.example         ← Configuration template
├── go.mod
└── go.sum
```

---

## 🌐 &nbsp; URL Reference

### Web Interface

| URL | Description |
|-----|-------------|
| `http://<ip>:8080/register` | Request access |
| `http://<ip>:8080/login` | Sign in |
| `http://<ip>:8080/` | Media library |
| `http://<ip>:8080/player?id=<id>` | Video player |
| `http://localhost:8080/admin` | Admin panel (localhost only) |

### REST API

**Auth (public)**

| Method | Route | Description |
|--------|-------|-------------|
| POST | `/api/auth/register` | Request access |
| POST | `/api/auth/login` | Login — returns token if approved |
| GET | `/api/auth/status?request_id=<id>` | Check request status |

**Media (token required)**

Token can be passed as a cookie (`magenta_token`), query param (`?token=`), or header (`Authorization: Bearer`).

| Method | Route | Description |
|--------|-------|-------------|
| GET | `/api/folders` | Media grouped by folder |
| GET | `/api/media` | Full media list |
| GET | `/api/media/{id}/next` | Next episode in folder |
| GET | `/api/playlist?folder=<path>` | M3U playlist for a folder |
| GET | `/stream/{id}` | Stream file (range requests) |
| GET | `/stream/{id}/vlc` | Download `.m3u` for VLC |

**Admin (localhost only)**

| Method | Route | Description |
|--------|-------|-------------|
| GET | `/admin/users` | List all users |
| POST | `/admin/users/{id}/approve` | Approve a user |
| POST | `/admin/users/{id}/revoke` | Revoke access |
| DELETE | `/admin/users/{id}` | Delete a user |

---

## 🧰 &nbsp; Tech Stack

<br/>

<div align="center">

| Layer | Technology |
|---|---|
| Language | Go 1.21+ |
| HTTP Server | `net/http` (stdlib) |
| Auth | bcrypt + secure random tokens |
| Persistence | JSON file (atomic writes) |
| File Watching | fsnotify |
| mDNS | grandcat/zeroconf |
| TLS | Self-signed via `crypto/x509` |
| Frontend | Vanilla HTML/CSS/JS (embedded) |
| Config | YAML (gopkg.in/yaml.v3) |
| Logging | `log/slog` (stdlib) |

</div>

---

## 🤝 &nbsp; Contributing

Contributions, suggestions, and bug reports are welcome.

1. Fork the repository
2. Create a feature branch: `git checkout -b feat/your-feature`
3. Implement and test your changes
4. Commit: `git commit -m "feat: describe your change"`
5. Push and open a Pull Request

---

## 📄 &nbsp; License

This project is licensed under the **GNU General Public License v3.0** — see the [`LICENSE`](LICENSE) file for details.

**Key Points:**
- ✅ You can use, modify, and distribute this software freely
- ✅ You must share any modifications under GPL v3
- ✅ You must include the original copyright and license notices
- ❌ You cannot make this software proprietary

Learn more at [gnu.org/licenses/gpl-3.0](https://www.gnu.org/licenses/gpl-3.0.html)

---

<div align="center">

<br/>

**MAGENTA**

*Your media. Your network. Your rules.*

<sub>Built with 🐹 Go · 🎬 HTTP Streaming · 📡 mDNS</sub>

<br/>

</div>
