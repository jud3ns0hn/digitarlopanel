# DigitarloPanel

Ein freies, umfangreiches Server-Administrations-Panel im Stil von aapanel —
**ohne Paywall**, ohne gesperrte Funktionen. Ausgeliefert als **eine einzige
Binärdatei** mit eingebettetem Web-Frontend.

## Funktionen (aktueller Stand)

- **Authentifizierung** — Login mit JWT, bcrypt-Passwörter, automatisch
  generierter Admin-Account beim ersten Start, Login-Rate-Limit, Audit-Log.
- **Dashboard** — Live-Monitoring (CPU, RAM, Load, Festplatten, Netzwerk,
  Uptime) per WebSocket mit ECharts-Diagrammen.
- **Datei-Manager** — Verzeichnisse durchsuchen, anlegen, umbenennen, löschen,
  hoch-/herunterladen, Text-Editor, Rechte (chmod) — mit Path-Traversal-Schutz.
- **Software & Dienste** — Cross-Distro-Installation und systemd-Steuerung von
  Nginx, MariaDB, Redis, PHP-FPM (apt bzw. dnf/yum).
- **Websites** — Nginx-vhosts anlegen, aktivieren/deaktivieren, löschen.
- **Datenbanken** — MySQL/MariaDB-Datenbanken und -Benutzer verwalten.
- **Cron-Jobs** — geplante Aufgaben verwalten (synchronisiert nach `/etc/cron.d`).

### Geplant (nächste Iterationen)

SSL / Let's Encrypt · Web-Terminal · Firewall-UI · Backups · RBAC / mehrere
Benutzer · 2FA.

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
| `listen`     | Bind-Adresse, Standard `:8088`                             |
| `data_dir`   | Speicherort der SQLite-DB, Standard `/var/lib/digitarlopanel` |
| `jwt_secret` | wird beim ersten Start zufällig erzeugt                    |
| `file_root`  | Wurzel für den Datei-Manager (`/` = gesamtes Dateisystem)  |

## Sicherheitshinweise

- Läuft als root — nur in vertrauenswürdigen Umgebungen einsetzen.
- `file_root` einschränken, um den Datei-Manager zu begrenzen.
- Panel hinter einen Reverse-Proxy mit TLS stellen oder TLS-Terminierung
  ergänzen (auf der Roadmap).
