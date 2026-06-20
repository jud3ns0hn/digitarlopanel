# DigitarloPanel

Ein freies, umfangreiches Server-Administrations-Panel im Stil von aapanel —
**ohne Paywall**, ohne gesperrte Funktionen. Ausgeliefert als **eine einzige
Binärdatei** mit eingebettetem Web-Frontend.

## Funktionen (aktueller Stand)

### Sicherheit
- **Authentifizierung** — JWT, bcrypt, automatisch generierter Admin beim
  ersten Start.
- **2FA (TOTP)** — Einrichtung per Authenticator-App, erzwungen beim Login.
- **RBAC** — Rollen admin / operator / viewer mit serverseitiger Durchsetzung.
- **Token-Revocation** — sofortiges Invalidieren aller Tokens (Logout,
  Passwort-/Rollenwechsel) über eine Token-Version im JWT.
- **Brute-Force-Schutz** — Account-Lockout nach Fehlversuchen, Login-Rate-Limit,
  zeitkonstante Benutzersuche gegen Enumeration.
- **Härtung** — strikte Security-Header inkl. Content-Security-Policy auf jeder
  Antwort; optionales Panel-TLS mit Auto-Self-Signed.
- **Audit-Log** — protokolliert alle sicherheitsrelevanten Aktionen.

### Verwaltung
- **Dashboard** — Live-Monitoring (CPU, RAM, Load, Disk, Netz, Uptime) per
  WebSocket mit ECharts, plus Top-Prozesse.
- **Datei-Manager** — Browsen, anlegen/umbenennen/löschen, Up-/Download,
  Editor, chmod, **Zippen/Entpacken** (Zip-Slip-geschützt) — mit
  Path-Traversal-Schutz.
- **Software & Dienste** — Cross-Distro-Installation und systemd-Steuerung
  (Nginx, MariaDB, Redis, PHP-FPM); zusätzlich generische systemd-Unit-Verwaltung.
- **Websites** — Nginx-vhosts anlegen/aktivieren/löschen, **PHP-Version pro
  Site**, **Reverse-Proxy** (App/Docker hinter Nginx, inkl. WebSocket).
- **PHP** — installierte PHP-FPM-Versionen erkennen und installieren.
- **Datenbanken** — MySQL/MariaDB-Datenbanken und -Benutzer verwalten.
- **SSL** — Let's-Encrypt-Zertifikate (ACME HTTP-01) und Self-Signed je Domain,
  mit automatischer Nginx-HTTPS-Konfiguration (HSTS, TLS 1.2/1.3).
- **Cron-Jobs** — geplante Aufgaben (Sync nach `/etc/cron.d`).
- **Docker** — Container auflisten/steuern, Images auflisten/ziehen.
- **Firewall** — ufw/firewalld: Status, Ports freigeben/sperren.
- **Backups** — tar.gz von Dateien und mysqldump von Datenbanken, plus
  **geplante Backups** (interner Cron-Scheduler mit Aufbewahrung).
- **Logs** — journalctl je Unit und Tail von Dateien unter `/var/log`.
- **Web-Terminal** — PTY-Shell über WebSocket (xterm.js), nur Admin, auditiert.
- **Benutzerverwaltung** — Benutzer/Rollen anlegen und verwalten (nur Admin).

### Geplant (nächste Iterationen)

FTP-Konten · persistente Monitoring-Historie · DNS-/Mailserver-Verwaltung.

## Architektur

- **Backend:** Go (Gin, GORM + SQLite, gopsutil, gorilla/websocket).
  Single-Binary über `go:embed` — keine Runtime-Abhängigkeiten auf dem Zielserver.
- **Frontend:** Vue 3 + Vite + TypeScript + Element Plus + ECharts.
- **Cross-Distro:** `osinfo` (Distro-Erkennung über `/etc/os-release`),
  `pkgmgr` (apt/dnf/yum), `service` (systemd). Unterstützt **Ubuntu/Debian** und
  **CentOS/RHEL/Rocky**.

```
backend/    Go-Server, API, Cross-Distro-Pakete, Embed
frontend/   Vue-3-SPA (baut nach backend/web/dist)
scripts/    install.sh (systemd-Installer)
```

## Build

Voraussetzungen: Go ≥ 1.24, Node ≥ 20.

```bash
make build        # baut Frontend + Backend → ./digitarlopanel
```

Oder einzeln:

```bash
make frontend     # Vue-Build nach backend/web/dist
make backend      # Go-Binary mit eingebettetem Frontend
make test         # Backend-Unit-Tests
```

## Start

```bash
sudo ./digitarlopanel -listen :8088
```

Beim ersten Start werden Admin-Benutzer und ein zufälliges Passwort in der
Konsole ausgegeben. Panel öffnen unter `http://<server>:8088`.

> Das Panel muss als **root** laufen, um Systemdienste, Pakete, Nginx-Configs
> und Datenbanken verwalten zu können. Passwort nach dem ersten Login ändern.

## Installation als Dienst

```bash
make build
sudo bash scripts/install.sh ./digitarlopanel
# Admin-Passwort anzeigen:
journalctl -u digitarlopanel --no-pager | grep -A2 'first run'
```

## Entwicklung

```bash
# Terminal 1 — Backend
make backend && ./digitarlopanel -config ./config.dev.json -listen :8088
# Terminal 2 — Frontend mit Hot-Reload (Proxy auf :8088)
cd frontend && npm run dev
```

## Konfiguration

Beim ersten Start wird `config.json` erzeugt:

| Feld         | Bedeutung                                                  |
|--------------|------------------------------------------------------------|
| `listen`        | Bind-Adresse, Standard `:8088`                          |
| `data_dir`      | Speicherort der SQLite-DB, Standard `/var/lib/digitarlopanel` |
| `jwt_secret`    | wird beim ersten Start zufällig erzeugt                 |
| `file_root`     | Wurzel für den Datei-Manager (`/` = gesamtes Dateisystem) |
| `backup_dir`    | Zielverzeichnis für Backups, Standard `/var/backups/digitarlopanel` |
| `tls_enabled`   | HTTPS fürs Panel aktivieren                             |
| `tls_auto_self_signed` | Self-Signed-Zertifikat automatisch erzeugen      |
| `tls_cert` / `tls_key` | Pfade zu eigenem Zertifikat/Schlüssel (optional)  |

Für HTTPS direkt im Panel: `tls_enabled` und `tls_auto_self_signed` auf `true`
setzen — beim Start wird ein Self-Signed-Zertifikat im `data_dir` erzeugt.

## Sicherheitshinweise

- Läuft als root — nur in vertrauenswürdigen Umgebungen einsetzen.
- `file_root` einschränken, um den Datei-Manager zu begrenzen.
- Panel hinter einen Reverse-Proxy mit TLS stellen oder TLS-Terminierung
  ergänzen (auf der Roadmap).
