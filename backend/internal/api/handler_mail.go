package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/mail"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/runner"
)

var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func (s *Server) handleMailDomainList(c *gin.Context) {
	var domains []model.MailDomain
	if err := s.db.Order("id desc").Find(&domains).Error; err != nil {
		serverError(c, err)
		return
	}
	var accounts []model.MailAccount
	s.db.Order("id desc").Find(&accounts)
	c.JSON(http.StatusOK, gin.H{"domains": domains, "accounts": accounts})
}

type mailDomainRequest struct {
	Domain string `json:"domain" binding:"required"`
}

func (s *Server) handleMailDomainCreate(c *gin.Context) {
	var req mailDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "domain required")
		return
	}
	if !domainPattern.MatchString(req.Domain) {
		badRequest(c, "invalid domain")
		return
	}
	d := model.MailDomain{Domain: req.Domain}
	if err := s.db.Create(&d).Error; err != nil {
		badRequest(c, "domain already exists")
		return
	}
	if err := s.syncMail(c); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "mail_domain_create", req.Domain)
	c.JSON(http.StatusOK, d)
}

func (s *Server) handleMailDomainDelete(c *gin.Context) {
	var d model.MailDomain
	if err := s.db.First(&d, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	s.db.Where("domain = ?", d.Domain).Delete(&model.MailAccount{})
	if err := s.db.Delete(&d).Error; err != nil {
		serverError(c, err)
		return
	}
	if err := s.syncMail(c); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "mail_domain_delete", d.Domain)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type mailAccountRequest struct {
	Address  string `json:"address" binding:"required"`
	Password string `json:"password" binding:"required"`
	Quota    int    `json:"quota"`
}

func (s *Server) handleMailAccountCreate(c *gin.Context) {
	var req mailAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "address and password required")
		return
	}
	if !emailPattern.MatchString(req.Address) {
		badRequest(c, "invalid email address")
		return
	}
	if len(req.Password) < 8 {
		badRequest(c, "password must be at least 8 characters")
		return
	}
	domain := req.Address[strings.Index(req.Address, "@")+1:]
	var md model.MailDomain
	if err := s.db.Where("domain = ?", domain).First(&md).Error; err != nil {
		badRequest(c, "create the mail domain first")
		return
	}

	hash, err := s.hashMailPassword(c.Request.Context(), req.Password)
	if err != nil {
		serverError(c, err)
		return
	}
	acc := model.MailAccount{Address: req.Address, Domain: domain, PasswordHash: hash, Quota: req.Quota}
	if err := s.db.Create(&acc).Error; err != nil {
		badRequest(c, "address already exists")
		return
	}
	if err := s.syncMail(c); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "mail_account_create", req.Address)
	c.JSON(http.StatusOK, acc)
}

func (s *Server) handleMailAccountDelete(c *gin.Context) {
	var acc model.MailAccount
	if err := s.db.First(&acc, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	if err := s.db.Delete(&acc).Error; err != nil {
		serverError(c, err)
		return
	}
	if err := s.syncMail(c); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "mail_account_delete", acc.Address)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// hashMailPassword produces a Dovecot-compatible SHA512-CRYPT hash using openssl.
func (s *Server) hashMailPassword(ctx context.Context, password string) (string, error) {
	ir, ok := s.runner.(runner.InputRunner)
	if !ok {
		return "", fmt.Errorf("runner does not support stdin")
	}
	res, err := ir.RunInput(ctx, password+"\n", "openssl", "passwd", "-6", "-stdin")
	if err != nil {
		return "", fmt.Errorf("openssl passwd: %w", err)
	}
	crypt := strings.TrimSpace(res.Stdout)
	if crypt == "" {
		return "", fmt.Errorf("empty hash from openssl")
	}
	return "{SHA512-CRYPT}" + crypt, nil
}

// syncMail regenerates Postfix/Dovecot maps and reloads the services.
func (s *Server) syncMail(c *gin.Context) error {
	var domains []model.MailDomain
	if err := s.db.Find(&domains).Error; err != nil {
		return err
	}
	var accounts []model.MailAccount
	if err := s.db.Find(&accounts).Error; err != nil {
		return err
	}
	domainNames := make([]string, 0, len(domains))
	for _, d := range domains {
		domainNames = append(domainNames, d.Domain)
	}
	mailAccounts := make([]mail.Account, 0, len(accounts))
	for _, a := range accounts {
		mailAccounts = append(mailAccounts, mail.Account{
			Address: a.Address, Domain: a.Domain, PasswordHash: a.PasswordHash,
		})
	}
	postmapTarget, err := mail.Sync(domainNames, mailAccounts)
	if err != nil {
		return err
	}
	ctx := c.Request.Context()
	if postmapTarget != "" {
		_, _ = s.runner.Run(ctx, "postmap", postmapTarget)
		_, _ = s.service.Reload(ctx, "postfix")
	}
	_, _ = s.service.Reload(ctx, "dovecot")
	return nil
}
