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
	ID         uint      `gorm:"primaryKey" json:"id"`
	Domain     string    `gorm:"uniqueIndex;size:255;not null" json:"domain"`
	Root       string    `gorm:"size:512;not null" json:"root"`
	PHPVersion string    `gorm:"size:16" json:"php_version"`
	Enabled    bool      `gorm:"default:true" json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
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
	}
}
