package api

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/certgen"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/nginx"
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

	cert := s.upsertCert(req.Domain, "letsencrypt", issued.CertPath, issued.KeyPath)
	if err := s.applyCertToSite(c, site, issued.CertPath, issued.KeyPath); err != nil {
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

	cert := s.upsertCert(req.Domain, "selfsigned", certPath, keyPath)

	// Bind to the website if one exists.
	var site model.Website
	if err := s.db.Where("domain = ?", req.Domain).First(&site).Error; err == nil {
		if err := s.applyCertToSite(c, site, certPath, keyPath); err != nil {
			serverError(c, err)
			return
		}
	}
	s.audit(c, "ssl_self_signed", req.Domain)
	c.JSON(http.StatusOK, cert)
}

func (s *Server) handleSSLDelete(c *gin.Context) {
	var cert model.Certificate
	if err := s.db.First(&cert, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "certificate not found"})
		return
	}
	// Rewrite the site without SSL if it exists.
	var site model.Website
	if err := s.db.Where("domain = ?", cert.Domain).First(&site).Error; err == nil {
		_ = nginx.Write(s.os.Family, nginx.VHost{Domain: site.Domain, Root: site.Root})
		_, _ = s.service.Reload(c.Request.Context(), "nginx")
	}
	if err := s.db.Delete(&cert).Error; err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "ssl_delete", cert.Domain)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// upsertCert stores or updates a certificate record, reading its expiry.
func (s *Server) upsertCert(domain, typ, certPath, keyPath string) model.Certificate {
	cert := model.Certificate{Domain: domain, Type: typ, CertPath: certPath, KeyPath: keyPath}
	if exp, err := certNotAfter(certPath); err == nil {
		cert.NotAfter = &exp
	}
	var existing model.Certificate
	if err := s.db.Where("domain = ?", domain).First(&existing).Error; err == nil {
		cert.ID = existing.ID
		cert.CreatedAt = existing.CreatedAt
	}
	s.db.Save(&cert)
	return cert
}

// applyCertToSite rewrites the site's vhost with the certificate and reloads nginx.
func (s *Server) applyCertToSite(c *gin.Context, site model.Website, certPath, keyPath string) error {
	if err := nginx.Write(s.os.Family, nginx.VHost{
		Domain:   site.Domain,
		Root:     site.Root,
		CertPath: certPath,
		KeyPath:  keyPath,
	}); err != nil {
		return err
	}
	_, _ = s.service.Reload(c.Request.Context(), "nginx")
	return nil
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
