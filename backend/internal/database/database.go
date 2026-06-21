package database

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"github.com/jud3ns0hn/digitarlopanel/backend/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Init opens the SQLite database, runs migrations and seeds the admin account.
// It returns the gorm handle and the generated admin password (empty if the
// admin already existed).
func Init(dbPath string) (*gorm.DB, string, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, "", fmt.Errorf("open database: %w", err)
	}

	if err := db.AutoMigrate(model.AllModels()...); err != nil {
		return nil, "", fmt.Errorf("migrate: %w", err)
	}

	password, err := seedAdmin(db)
	if err != nil {
		return nil, "", err
	}
	return db, password, nil
}

// seedAdmin creates a default admin user on first run and returns its
// randomly generated password. Returns an empty password if one already exists.
func seedAdmin(db *gorm.DB) (string, error) {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return "", err
	}
	if count > 0 {
		return "", nil
	}

	password, err := randomPassword(16)
	if err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	admin := &model.User{
		Username:     "admin",
		PasswordHash: string(hash),
		Role:         "admin",
	}
	if err := db.Create(admin).Error; err != nil {
		return "", err
	}
	log.Printf("Created default admin account (username: admin)")
	return password, nil
}
