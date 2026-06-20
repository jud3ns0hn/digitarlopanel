// Package acme issues Let's Encrypt certificates via the HTTP-01 challenge
// using a webroot, and persists the account key between runs.
package acme

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/providers/http/webroot"
	"github.com/go-acme/lego/v4/registration"
)

// user implements lego's registration.User backed by a persisted account key.
type user struct {
	email        string
	registration *registration.Resource
	key          crypto.PrivateKey
}

func (u *user) GetEmail() string                        { return u.email }
func (u *user) GetRegistration() *registration.Resource { return u.registration }
func (u *user) GetPrivateKey() crypto.PrivateKey        { return u.key }

// Issued holds the PEM material returned by a successful order.
type Issued struct {
	CertPath string
	KeyPath  string
}

// Issuer obtains certificates for domains and writes them under certDir.
type Issuer struct {
	accountDir string // where the ACME account key lives
	certDir    string // where issued certs are written
	caURL      string // ACME directory URL
}

// NewIssuer creates an Issuer. If staging is true the Let's Encrypt staging CA
// is used (recommended while testing to avoid rate limits).
func NewIssuer(accountDir, certDir string, staging bool) *Issuer {
	ca := lego.LEDirectoryProduction
	if staging {
		ca = lego.LEDirectoryStaging
	}
	return &Issuer{accountDir: accountDir, certDir: certDir, caURL: ca}
}

// Obtain runs the ACME flow for the given domains using an HTTP-01 challenge
// served from webrootPath. It returns the paths of the written cert and key.
func (i *Issuer) Obtain(email string, domains []string, webrootPath string) (Issued, error) {
	if len(domains) == 0 {
		return Issued{}, fmt.Errorf("no domains given")
	}
	key, err := i.loadOrCreateAccountKey(email)
	if err != nil {
		return Issued{}, err
	}

	u := &user{email: email, key: key}
	cfg := lego.NewConfig(u)
	cfg.CADirURL = i.caURL
	cfg.Certificate.KeyType = "ec256"

	client, err := lego.NewClient(cfg)
	if err != nil {
		return Issued{}, err
	}

	provider, err := webroot.NewHTTPProvider(webrootPath)
	if err != nil {
		return Issued{}, err
	}
	if err := client.Challenge.SetHTTP01Provider(provider); err != nil {
		return Issued{}, err
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return Issued{}, fmt.Errorf("register account: %w", err)
	}
	u.registration = reg

	res, err := client.Certificate.Obtain(certificate.ObtainRequest{Domains: domains, Bundle: true})
	if err != nil {
		return Issued{}, fmt.Errorf("obtain certificate: %w", err)
	}

	if err := os.MkdirAll(i.certDir, 0o750); err != nil {
		return Issued{}, err
	}
	certPath := filepath.Join(i.certDir, domains[0]+".crt")
	keyPath := filepath.Join(i.certDir, domains[0]+".key")
	if err := os.WriteFile(certPath, res.Certificate, 0o644); err != nil {
		return Issued{}, err
	}
	if err := os.WriteFile(keyPath, res.PrivateKey, 0o600); err != nil {
		return Issued{}, err
	}
	return Issued{CertPath: certPath, KeyPath: keyPath}, nil
}

// loadOrCreateAccountKey returns a persisted ECDSA account key for the email,
// creating one on first use.
func (i *Issuer) loadOrCreateAccountKey(email string) (crypto.PrivateKey, error) {
	if err := os.MkdirAll(i.accountDir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(i.accountDir, "account-"+sanitize(email)+".key")

	if data, err := os.ReadFile(path); err == nil {
		block, _ := pem.Decode(data)
		if block == nil {
			return nil, fmt.Errorf("invalid account key file")
		}
		return x509.ParseECPrivateKey(block.Bytes)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	pemData := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, pemData, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

// sanitize makes an email safe for use in a filename.
func sanitize(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}
	return string(out)
}
