package api

import (
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/pkg/dns"
)

var recordTypePattern = regexp.MustCompile(`^(A|AAAA|CNAME|MX|TXT|NS)$`)
var recordValuePattern = regexp.MustCompile(`^[a-zA-Z0-9 ._:/"@.\-]{1,512}$`)

func (s *Server) handleDNSZoneList(c *gin.Context) {
	var zones []model.DNSZone
	if err := s.db.Preload("Records").Order("id desc").Find(&zones).Error; err != nil {
		serverError(c, err)
		return
	}
	c.JSON(http.StatusOK, zones)
}

type dnsZoneCreateRequest struct {
	Domain string `json:"domain" binding:"required"`
	NS     string `json:"ns"`
	Admin  string `json:"admin"`
}

func (s *Server) handleDNSZoneCreate(c *gin.Context) {
	var req dnsZoneCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "domain required")
		return
	}
	if !domainPattern.MatchString(req.Domain) {
		badRequest(c, "invalid domain")
		return
	}
	zone := model.DNSZone{
		Domain: req.Domain, NS: req.NS, Admin: req.Admin,
		Serial: uint32(time.Now().Unix()),
	}
	if err := s.db.Create(&zone).Error; err != nil {
		badRequest(c, "zone already exists")
		return
	}
	if err := s.syncDNS(c); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "dns_zone_create", req.Domain)
	c.JSON(http.StatusOK, zone)
}

func (s *Server) handleDNSZoneDelete(c *gin.Context) {
	var zone model.DNSZone
	if err := s.db.First(&zone, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "zone not found"})
		return
	}
	s.db.Where("zone_id = ?", zone.ID).Delete(&model.DNSRecord{})
	if err := s.db.Delete(&zone).Error; err != nil {
		serverError(c, err)
		return
	}
	if err := s.syncDNS(c); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "dns_zone_delete", zone.Domain)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type dnsRecordRequest struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Value    string `json:"value" binding:"required"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
}

func (s *Server) handleDNSRecordCreate(c *gin.Context) {
	var zone model.DNSZone
	if err := s.db.First(&zone, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "zone not found"})
		return
	}
	var req dnsRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "name, type and value required")
		return
	}
	if !recordTypePattern.MatchString(req.Type) {
		badRequest(c, "type must be A, AAAA, CNAME, MX, TXT or NS")
		return
	}
	if !recordValuePattern.MatchString(req.Value) {
		badRequest(c, "invalid record value")
		return
	}
	if req.TTL <= 0 {
		req.TTL = 3600
	}
	rec := model.DNSRecord{
		ZoneID: zone.ID, Name: req.Name, Type: req.Type,
		Value: req.Value, TTL: req.TTL, Priority: req.Priority,
	}
	if err := s.db.Create(&rec).Error; err != nil {
		serverError(c, err)
		return
	}
	s.bumpSerial(&zone)
	if err := s.syncDNS(c); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "dns_record_create", zone.Domain+" "+req.Type+" "+req.Name)
	c.JSON(http.StatusOK, rec)
}

func (s *Server) handleDNSRecordDelete(c *gin.Context) {
	var rec model.DNSRecord
	if err := s.db.First(&rec, c.Param("rid")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "record not found"})
		return
	}
	if err := s.db.Delete(&rec).Error; err != nil {
		serverError(c, err)
		return
	}
	var zone model.DNSZone
	if err := s.db.First(&zone, rec.ZoneID).Error; err == nil {
		s.bumpSerial(&zone)
	}
	if err := s.syncDNS(c); err != nil {
		serverError(c, err)
		return
	}
	s.audit(c, "dns_record_delete", rec.Type+" "+rec.Name)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// bumpSerial advances the zone serial so secondaries notice the change.
func (s *Server) bumpSerial(zone *model.DNSZone) {
	zone.Serial = uint32(time.Now().Unix())
	s.db.Save(zone)
}

// syncDNS regenerates all zone files and reloads BIND.
func (s *Server) syncDNS(c *gin.Context) error {
	var zones []model.DNSZone
	if err := s.db.Preload("Records").Find(&zones).Error; err != nil {
		return err
	}
	out := make([]dns.Zone, 0, len(zones))
	for _, z := range zones {
		records := make([]dns.Record, 0, len(z.Records))
		for _, r := range z.Records {
			records = append(records, dns.Record{
				Name: r.Name, Type: r.Type, Value: r.Value, TTL: r.TTL, Priority: r.Priority,
			})
		}
		out = append(out, dns.Zone{
			Domain: z.Domain, NS: z.NS, Admin: z.Admin, Serial: z.Serial, Records: records,
		})
	}
	if err := dns.Sync(s.os.Family, out); err != nil {
		return err
	}
	_, _ = s.service.Reload(c.Request.Context(), dns.ReloadUnit(s.os.Family))
	return nil
}
