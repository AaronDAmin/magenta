# Changelog

All notable changes to Magenta will be documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

### Planned
- Subtitle support (.srt, .vtt)
- Mobile-optimized UI
- Watch progress persistence

---

## [1.0.0] — Initial Release

### Added
- Netflix-style web interface (dark theme, card grid, search)
- Built-in video player with keyboard shortcuts (Space, ←/→, F, M)
- Series support — episodes auto-linked, autoplay next
- Approval-based auth — register → admin approves → token issued
- HTTP range request streaming (seek without full buffering)
- VLC integration — one-click `.m3u` launch on PC/Mac/Android
- M3U playlist generation for entire folders
- Auto-scan on startup across multiple directories
- Live file watching via fsnotify
- Multi-root path security (whitelisted directories only)
- mDNS discovery (`http://magenta.local:8080`)
- Optional TLS/HTTPS with self-signed certificate generation
- Admin panel (localhost only) — approve, revoke, delete users
- Single Go binary — embeds all HTML/CSS assets at build time
- Structured logging via `log/slog`
- Graceful shutdown on SIGINT/SIGTERM
