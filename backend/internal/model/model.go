package model

import "time"

// User is a panel account.
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	Role         string    `gorm:"size:32;default:admin" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Website is an Nginx virtual host managed by the panel.
type Website struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Domain    string    `gorm:"uniqueIndex;size:255;not null" json:"domain"`
	Root      string    `gorm:"size:512;not null" json:"root"`
	PHPVersion string   `gorm:"size:16" json:"php_version"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	}
}
