package api

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/certgen"
)

func (s *Server) handleSSLList(c *gin.Context) {
	var certs []model.Certificate
	if err := s.db.Order("id desc").Find(&certs).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, certs)
}

type sslIssueRequest struct {
	Domain string `json:"domain" binding:"required"`
	Email  string `json:"email" binding:"required"`
}

// handleSSLIssue obtains a Let's Encrypt certificate via HTTP-01. The website
// for the domain must already exist so the ACME challenge can be served from
// its document root.
func (s *Server) handleSSLIssue(c *gin.Context) {
	var req sslIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "domain and email required")
		return
	}
	if !domainPattern.MatchString(req.Domain) {
		badRequest(c, "invalid domain")
		return
	}

	var site model.Website
	if err := s.db.Where("domain = ?", req.Domain).First(&site).Error; err != nil {
		badRequest(c, "create the website first so the ACME challenge can be served")
		return
	}

	issued, err := s.issuer.Obtain(req.Email, []string{req.Domain}, site.Root)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cert := s.upsertCert(req.Domain, "letsencrypt", req.Email, issued.CertPath, issued.KeyPath)
	if err := s.writeSiteVHost(c.Request.Context(), site); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "ssl_issue", req.Domain)
	c.JSON(http.StatusOK, cert)
}

type sslSelfSignedRequest struct {
	Domain string `json:"domain" binding:"required"`
}

// handleSSLSelfSigned generates a self-signed certificate and binds it to the
// website, enabling HTTPS immediately (browsers will warn until trusted).
func (s *Server) handleSSLSelfSigned(c *gin.Context) {
	var req sslSelfSignedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "domain required")
		return
	}
	if !domainPattern.MatchString(req.Domain) {
		badRequest(c, "invalid domain")
		return
	}
	if err := os.MkdirAll(s.certDir, 0o750); err != nil {
		serverError(c, err)
		return
	}
	certPath := filepath.Join(s.certDir, req.Domain+".crt")
	keyPath := filepath.Join(s.certDir, req.Domain+".key")
	if err := certgen.GenerateForDomain(certPath, keyPath, req.Domain); err != nil {
		serverError(c, err)
		return
	}

	cert := s.upsertCert(req.Domain, "selfsigned", "", certPath, keyPath)

	// Bind to the website if one exists.
	var site model.Website
	if err := s.db.Where("domain = ?", req.Domain).First(&site).Error; err == nil {
		if err := s.writeSiteVHost(c.Request.Context(), site); err != nil {
			serverError(c, err)
			return
		}
	}
	s.audit(c, "ssl_self_signed", req.Domain)
	c.JSON(http.StatusOK, cert)
}

// handleSSLRenew re-issues a Let's Encrypt certificate on demand.
func (s *Server) handleSSLRenew(c *gin.Context) {
	var cert model.Certificate
	if err := s.db.First(&cert, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
		return
	}
	if cert.Type != "letsencrypt" || cert.Email == "" {
		badRequest(c, "only Let's Encrypt certificates with a stored e-mail can be renewed")
		return
	}
	var site model.Website
	if err := s.db.Where("domain = ?", cert.Domain).First(&site).Error; err != nil {
		badRequest(c, "website for this domain no longer exists")
		return
	}
	issued, err := s.issuer.Obtain(cert.Email, []string{cert.Domain}, site.Root)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	updated := s.upsertCert(cert.Domain, "letsencrypt", cert.Email, issued.CertPath, issued.KeyPath)
	if err := s.writeSiteVHost(c.Request.Context(), site); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "ssl_renew", cert.Domain)
	c.JSON(http.StatusOK, updated)
}

func (s *Server) handleSSLDelete(c *gin.Context) {
	var cert model.Certificate
	if err := s.db.First(&cert, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
		return
	}
	// Delete first so the rewrite below picks up the absence of the cert.
	if err := s.db.Delete(&cert).Error; err != nil {
		serverError(c, err)
		return
	}
	// Rewrite the site without SSL if it exists (PHP settings are preserved).
	var site model.Website
	if err := s.db.Where("domain = ?", cert.Domain).First(&site).Error; err == nil {
		_ = s.writeSiteVHost(c.Request.Context(), site)
	}
	s.audit(c, "ssl_delete", cert.Domain)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// upsertCert stores or updates a certificate record, reading its expiry.
func (s *Server) upsertCert(domain, typ, email, certPath, keyPath string) model.Certificate {
	cert := model.Certificate{Domain: domain, Type: typ, Email: email, CertPath: certPath, KeyPath: keyPath}
	if exp, err := certNotAfter(certPath); err == nil {
		cert.NotAfter = &exp
	}
	var existing model.Certificate
	if err := s.db.Where("domain = ?", domain).First(&existing).Error; err == nil {
		cert.ID = existing.ID
		cert.CreatedAt = existing.CreatedAt
		if email == "" {
			cert.Email = existing.Email
		}
	}
	s.db.Save(&cert)
	return cert
}

// renewExpiringCerts re-issues Let's Encrypt certificates within 30 days of
// expiry. Called daily by the scheduler.
func (s *Server) renewExpiringCerts(ctx context.Context) {
	var certs []model.Certificate
	s.db.Where("type = ?", "letsencrypt").Find(&certs)
	cutoff := time.Now().Add(30 * 24 * time.Hour)
	for _, cert := range certs {
		if cert.NotAfter == nil || cert.NotAfter.After(cutoff) || cert.Email == "" {
			continue
		}
		var site model.Website
		if err := s.db.Where("domain = ?", cert.Domain).First(&site).Error; err != nil {
			continue
		}
		issued, err := s.issuer.Obtain(cert.Email, []string{cert.Domain}, site.Root)
		if err != nil {
			continue
		}
		s.upsertCert(cert.Domain, "letsencrypt", cert.Email, issued.CertPath, issued.KeyPath)
		_ = s.writeSiteVHost(ctx, site)
	}
}

// certNotAfter parses a PEM certificate file and returns its expiry.
func certNotAfter(path string) (time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return time.Time{}, os.ErrInvalid
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, err
	}
	return cert.NotAfter, nil
}
