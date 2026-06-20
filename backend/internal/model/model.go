package model

import "time"

// User is a panel account.
type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string `gorm:"not null" json:"-"`
	Role         string `gorm:"size:32;default:admin" json:"role"`

	// TwoFAEnabled indicates an active TOTP second factor.
	TwoFAEnabled bool   `gorm:"default:false" json:"two_fa_enabled"`
	TwoFASecret  string `gorm:"size:128" json:"-"`

	// TokenVersion is embedded in issued JWTs; bumping it revokes all existing
	// tokens for this user (used by logout-everywhere and password changes).
	TokenVersion int `gorm:"default:0" json:"-"`

	// Brute-force protection.
	FailedAttempts int        `gorm:"default:0" json:"-"`
	LockedUntil    *time.Time `json:"locked_until,omitempty"`

	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP string     `gorm:"size:64" json:"last_login_ip,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Valid panel roles, from most to least privileged.
const (
	RoleAdmin    = "admin"    // full control, including user management
	RoleOperator = "operator" // manage services/sites/files, no user management
	RoleViewer   = "viewer"   // read-only
)

// Certificate records a TLS certificate managed by the panel.
type Certificate struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Domain    string     `gorm:"uniqueIndex;size:255;not null" json:"domain"`
	Type      string     `gorm:"size:32;not null" json:"type"` // letsencrypt | selfsigned
	Email     string     `gorm:"size:255" json:"email"`        // ACME account email (for renewal)
	CertPath  string     `gorm:"size:512" json:"cert_path"`
	KeyPath   string     `gorm:"size:512" json:"key_path"`
	NotAfter  *time.Time `json:"not_after,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Backup records an archive created by the panel.
type Backup struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Type      string    `gorm:"size:32;not null" json:"type"` // files | database
	Source    string    `gorm:"size:512" json:"source"`       // path or database name
	Path      string    `gorm:"size:512;not null" json:"path"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// ScheduledBackup defines a recurring backup run by the in-process scheduler.
type ScheduledBackup struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Name       string     `gorm:"size:128;not null" json:"name"`
	Type       string     `gorm:"size:32;not null" json:"type"`     // files | database
	Source     string     `gorm:"size:512;not null" json:"source"`  // path or database name
	Schedule   string     `gorm:"size:64;not null" json:"schedule"` // cron expression
	Retention  int        `gorm:"default:7" json:"retention"`       // keep N most recent
	Enabled    bool       `gorm:"default:true" json:"enabled"`
	LastRunAt  *time.Time `json:"last_run_at,omitempty"`
	LastStatus string     `gorm:"size:256" json:"last_status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Website is an Nginx virtual host managed by the panel.
type Website struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	Domain     string `gorm:"uniqueIndex;size:255;not null" json:"domain"`
	Root       string `gorm:"size:512;not null" json:"root"`
	PHPVersion string `gorm:"size:16" json:"php_version"`
	// ProxyPass, when set, turns the site into a reverse proxy to this upstream
	// (e.g. http://127.0.0.1:3000) instead of serving files.
	ProxyPass string `gorm:"size:255" json:"proxy_pass"`
	// Redirect, when set, makes the whole site issue a 301 to this URL.
	Redirect string `gorm:"size:255" json:"redirect"`
	// ExtraConfig is an admin-provided raw nginx snippet for the server block.
	ExtraConfig string `gorm:"size:4096" json:"extra_config"`
	// BasicAuthUser/Hash protect the site with HTTP Basic auth when set.
	BasicAuthUser string    `gorm:"size:64" json:"basic_auth_user"`
	BasicAuthHash string    `gorm:"size:255" json:"-"`
	Enabled       bool      `gorm:"default:true" json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DatabaseInstance records a MySQL/MariaDB database created via the panel.
type DatabaseInstance struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Username  string    `gorm:"size:64;not null" json:"username"`
	Charset   string    `gorm:"size:32;default:utf8mb4" json:"charset"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PostgresInstance records a PostgreSQL database created via the panel.
type PostgresInstance struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Username  string    `gorm:"size:64;not null" json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CronJob is a scheduled task synced to the host crontab.
type CronJob struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Schedule  string    `gorm:"size:64;not null" json:"schedule"` // cron expression, e.g. "0 3 * * *"
	Command   string    `gorm:"size:1024;not null" json:"command"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MetricSample is a periodic snapshot of host load persisted for history charts.
type MetricSample struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	Timestamp   int64   `gorm:"index" json:"timestamp"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemPercent  float64 `json:"mem_percent"`
	Load1       float64 `json:"load1"`
	DiskPercent float64 `json:"disk_percent"`
}

// AuditLog records a security-relevant action.
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `json:"user_id"`
	Username  string    `gorm:"size:64" json:"username"`
	Action    string    `gorm:"size:128" json:"action"`
	Detail    string    `gorm:"size:1024" json:"detail"`
	IP        string    `gorm:"size:64" json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}

// FTPAccount is a virtual FTP user managed via pure-ftpd.
type FTPAccount struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Home      string    `gorm:"size:512;not null" json:"home"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DNSZone is an authoritative DNS zone served by BIND.
type DNSZone struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	Domain    string      `gorm:"uniqueIndex;size:255;not null" json:"domain"`
	NS        string      `gorm:"size:255" json:"ns"`    // primary nameserver
	Admin     string      `gorm:"size:255" json:"admin"` // admin email (zone SOA)
	Serial    uint32      `json:"serial"`                // SOA serial
	Records   []DNSRecord `gorm:"foreignKey:ZoneID" json:"records,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// DNSRecord is a single resource record in a zone.
type DNSRecord struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	ZoneID   uint   `gorm:"index" json:"zone_id"`
	Name     string `gorm:"size:255" json:"name"` // "@" or subdomain label
	Type     string `gorm:"size:16" json:"type"`  // A, AAAA, CNAME, MX, TXT, NS
	Value    string `gorm:"size:512" json:"value"`
	TTL      int    `gorm:"default:3600" json:"ttl"`
	Priority int    `json:"priority"` // for MX
}

// MailDomain is a virtual mail domain.
type MailDomain struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Domain    string    `gorm:"uniqueIndex;size:255;not null" json:"domain"`
	CreatedAt time.Time `json:"created_at"`
}

// MailAccount is a virtual mailbox.
type MailAccount struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Address      string    `gorm:"uniqueIndex;size:320;not null" json:"address"`
	Domain       string    `gorm:"size:255;not null" json:"domain"`
	PasswordHash string    `gorm:"size:255" json:"-"`
	Quota        int       `gorm:"default:0" json:"quota"` // MB, 0 = unlimited
	CreatedAt    time.Time `json:"created_at"`
}

// ComposeApp is a docker-compose stack deployed by the panel.
type ComposeApp struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;size:64;not null" json:"name"`
	Dir       string    `gorm:"size:512;not null" json:"dir"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UptimeMonitor periodically checks an HTTP(S) endpoint.
type UptimeMonitor struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Name       string     `gorm:"size:128;not null" json:"name"`
	URL        string     `gorm:"size:512;not null" json:"url"`
	IntervalS  int        `gorm:"default:60" json:"interval_s"`
	Enabled    bool       `gorm:"default:true" json:"enabled"`
	LastStatus string     `gorm:"size:32" json:"last_status"` // up | down
	LastCode   int        `json:"last_code"`
	LastMS     int64      `json:"last_ms"`
	LastCheck  *time.Time `json:"last_check,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// AlertRule fires a notification when a metric crosses a threshold.
type AlertRule struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	Metric    string     `gorm:"size:32;not null" json:"metric"`  // cpu | memory | disk
	Threshold float64    `json:"threshold"`                       // percent
	Channel   string     `gorm:"size:32;not null" json:"channel"` // email | webhook
	Target    string     `gorm:"size:512;not null" json:"target"` // email address or webhook URL
	Enabled   bool       `gorm:"default:true" json:"enabled"`
	LastFired *time.Time `json:"last_fired,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// BackupDestination is a remote target for uploading backups.
type BackupDestination struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	Type      string    `gorm:"size:32;not null" json:"type"` // s3 | sftp | webdav
	Endpoint  string    `gorm:"size:512" json:"endpoint"`     // S3 endpoint / host / webdav url
	Bucket    string    `gorm:"size:255" json:"bucket"`       // S3 bucket / remote dir
	AccessKey string    `gorm:"size:255" json:"access_key"`   // S3 key / sftp user / webdav user
	SecretKey string    `gorm:"size:512" json:"-"`            // S3 secret / sftp password / webdav password
	Region    string    `gorm:"size:64" json:"region"`
	CreatedAt time.Time `json:"created_at"`
}

// AllModels returns every model for AutoMigrate.
func AllModels() []any {
	return []any{
		&User{},
		&Website{},
		&DatabaseInstance{},
		&CronJob{},
		&AuditLog{},
		&Backup{},
		&ScheduledBackup{},
		&Certificate{},
		&MetricSample{},
		&FTPAccount{},
		&DNSZone{},
		&DNSRecord{},
		&MailDomain{},
		&MailAccount{},
		&ComposeApp{},
		&PostgresInstance{},
		&UptimeMonitor{},
		&AlertRule{},
		&BackupDestination{},
	}
}
