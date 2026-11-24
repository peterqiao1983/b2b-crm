package db

import (
	"log"
	"os"
	"path/filepath"

	"b2b-crm/internal/models"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Init(dbPath string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	dsn := dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := database.DB()
	if err != nil {
		return nil, err
	}
	// 10 人以内并发使用单写连接串行化，避免 SQLite database is locked
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	if err := database.AutoMigrate(
		&models.User{},
		&models.Company{},
		&models.Contact{},
		&models.Project{},
		&models.ProjectContact{},
		&models.FollowUp{},
	); err != nil {
		return nil, err
	}

	seedDefaultAdmin(database)
	return database, nil
}

func seedDefaultAdmin(database *gorm.DB) {
	var count int64
	if err := database.Model(&models.User{}).Count(&count).Error; err != nil {
		log.Printf("count users failed: %v", err)
		return
	}
	if count > 0 {
		return
	}

	username := os.Getenv("CRM_INIT_USER")
	if username == "" {
		username = "peter"
	}
	password := os.Getenv("CRM_INIT_PASSWORD")
	if password == "" {
		password = "REDACTED"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("hash default password failed: %v", err)
		return
	}

	admin := models.User{
		Username:     username,
		PasswordHash: string(hash),
		DisplayName:  "Peter",
		Role:         "admin",
	}
	if err := database.Create(&admin).Error; err != nil {
		log.Printf("create default admin failed: %v", err)
	} else {
		log.Printf("initialized default admin user: %s", username)
	}
}
