package nginx

import "strings"

import "testing"

func TestRenderStatic(t *testing.T) {
	out, err := Render(VHost{Domain: "example.com", Root: "/var/www"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "server_name example.com;") {
		t.Error("missing server_name")
	}
	if strings.Contains(out, "listen 443") {
		t.Error("should not have HTTPS block without cert")
	}
	if strings.Contains(out, "fastcgi_pass") {
		t.Error("should not have php without socket")
	}
}

func TestRenderPHP(t *testing.T) {
	out, err := Render(VHost{Domain: "php.com", Root: "/srv", PHPSocket: "/run/php/php8.3-fpm.sock"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "fastcgi_pass unix:/run/php/php8.3-fpm.sock;") {
		t.Error("missing php fastcgi_pass")
	}
}

func TestRenderSSLAndPHP(t *testing.T) {
	out, err := Render(VHost{
		Domain: "secure.com", Root: "/srv",
		CertPath: "/c.crt", KeyPath: "/c.key",
		PHPSocket: "/run/php/php8.3-fpm.sock",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "listen 443 ssl;") {
		t.Error("missing HTTPS block")
	}
	if !strings.Contains(out, "return 301 https://") {
		t.Error("missing HTTP->HTTPS redirect")
	}
	if !strings.Contains(out, "fastcgi_pass") {
		t.Error("php should be present in HTTPS block")
	}
}

func TestRenderProxy(t *testing.T) {
	out, err := Render(VHost{Domain: "app.com", Root: "/srv", ProxyPass: "http://127.0.0.1:3000"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "proxy_pass http://127.0.0.1:3000;") {
		t.Error("missing proxy_pass")
	}
	if strings.Contains(out, "try_files") {
		t.Error("proxy site should not use try_files")
	}
}
