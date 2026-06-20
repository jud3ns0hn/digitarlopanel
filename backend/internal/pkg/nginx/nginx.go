// Package nginx renders and writes virtual host configuration files and
// reloads the nginx service.
package nginx

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/osinfo"
)

// VHost describes the inputs to a server block.
type VHost struct {
	Domain string
	Root   string
	// SSL fields; when CertPath and KeyPath are set, an HTTPS server block is
	// emitted and HTTP is redirected to HTTPS (except the ACME challenge path).
	CertPath string
	KeyPath  string
	// PHPSocket, when set, adds a FastCGI location passing .php requests to the
	// given php-fpm unix socket.
	PHPSocket string
	// ProxyPass, when set, makes the site a reverse proxy to this upstream URL.
	ProxyPass string
}

// SSLEnabled reports whether the vhost should serve HTTPS.
func (v VHost) SSLEnabled() bool { return v.CertPath != "" && v.KeyPath != "" }

// PHPEnabled reports whether a php-fpm backend is configured (ignored when the
// site is a reverse proxy).
func (v VHost) PHPEnabled() bool { return v.PHPSocket != "" && v.ProxyPass == "" }

// ProxyEnabled reports whether the site is a reverse proxy.
func (v VHost) ProxyEnabled() bool { return v.ProxyPass != "" }

const vhostTemplate = `# Managed by DigitarloPanel - do not edit by hand
server {
    listen 80;
    listen [::]:80;
    server_name {{ .Domain }};
    root {{ .Root }};
    index index.html index.htm index.php;

    access_log /var/log/nginx/{{ .Domain }}.access.log;
    error_log  /var/log/nginx/{{ .Domain }}.error.log;

    location ^~ /.well-known/acme-challenge/ {
        root {{ .Root }};
        allow all;
    }
{{- if .SSLEnabled }}

    location / {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    http2 on;
    server_name {{ .Domain }};
    root {{ .Root }};
    index index.html index.htm index.php;

    ssl_certificate     {{ .CertPath }};
    ssl_certificate_key {{ .KeyPath }};
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    add_header Strict-Transport-Security "max-age=31536000" always;

    access_log /var/log/nginx/{{ .Domain }}.access.log;
    error_log  /var/log/nginx/{{ .Domain }}.error.log;

{{- if .ProxyEnabled }}
    location / {
        proxy_pass {{ .ProxyPass }};
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
{{- else }}
    location / {
        try_files $uri $uri/ /index.php?$query_string /index.html;
    }
{{- if .PHPEnabled }}

    location ~ \.php$ {
        include fastcgi_params;
        fastcgi_pass unix:{{ .PHPSocket }};
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
    }
{{- end }}
{{- end }}

    location ~ /\.(?!well-known).* {
        deny all;
    }
}
{{- else }}

{{- if .ProxyEnabled }}
    location / {
        proxy_pass {{ .ProxyPass }};
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
{{- else }}
    location / {
        try_files $uri $uri/ /index.php?$query_string /index.html;
    }
{{- if .PHPEnabled }}

    location ~ \.php$ {
        include fastcgi_params;
        fastcgi_pass unix:{{ .PHPSocket }};
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
    }
{{- end }}
{{- end }}

    location ~ /\.(?!well-known).* {
        deny all;
    }
}
{{- end }}
`

var tmpl = template.Must(template.New("vhost").Parse(vhostTemplate))

// Render produces the nginx config text for a vhost.
func Render(v VHost) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, v); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// confDir returns the directory where vhost files live for the given family.
// Debian uses sites-available with a sites-enabled symlink; RHEL uses conf.d.
func confDir(family osinfo.Family) (available, enabled string) {
	switch family {
	case osinfo.FamilyDebian:
		return "/etc/nginx/sites-available", "/etc/nginx/sites-enabled"
	default:
		return "/etc/nginx/conf.d", "/etc/nginx/conf.d"
	}
}

// ConfPath returns the on-disk path of a vhost's primary config file.
func ConfPath(family osinfo.Family, domain string) string {
	available, _ := confDir(family)
	if family == osinfo.FamilyDebian {
		return filepath.Join(available, domain+".conf")
	}
	return filepath.Join(available, domain+".conf")
}

// Write renders and persists a vhost, creating the document root and (on
// Debian) the sites-enabled symlink. It does not reload nginx.
func Write(family osinfo.Family, v VHost) error {
	content, err := Render(v)
	if err != nil {
		return err
	}
	available, enabled := confDir(family)
	if err := os.MkdirAll(available, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(enabled, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(v.Root, 0o755); err != nil {
		return fmt.Errorf("create root: %w", err)
	}

	confPath := ConfPath(family, v.Domain)
	if err := os.WriteFile(confPath, []byte(content), 0o644); err != nil {
		return err
	}

	if family == osinfo.FamilyDebian && enabled != available {
		link := filepath.Join(enabled, v.Domain+".conf")
		_ = os.Remove(link)
		if err := os.Symlink(confPath, link); err != nil {
			return fmt.Errorf("enable site: %w", err)
		}
	}
	return nil
}

// Remove deletes a vhost's config file and (on Debian) its symlink.
func Remove(family osinfo.Family, domain string) error {
	available, enabled := confDir(family)
	_ = os.Remove(filepath.Join(enabled, domain+".conf"))
	if available != enabled {
		_ = os.Remove(filepath.Join(available, domain+".conf"))
	}
	return nil
}
