package db

import (
	"log"
	"os"
	"path/filepath"
	"time"

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
		&models.AuditLog{},
	); err != nil {
		return nil, err
	}
	for _, tbl := range []string{"companies", "contacts", "projects", "follow_ups"} {
		ensureDeletedAtColumn(database, tbl)
	}
	migrateContactNullableCompanyID(database)
	seedDefaultAdmin(database)
	go startWALCheckpointLoop(database)
	return database, nil
}

func migrateContactNullableCompanyID(database *gorm.DB) {
	type colInfo struct {
		Name    string `gorm:"column:name"`
		NotNull int    `gorm:"column:notnull"`
	}
	var cols []colInfo
	if err := database.Raw("PRAGMA table_info(contacts)").Scan(&cols).Error; err != nil {
		return
	}
	needMigrate := false
	for _, c := range cols {
		if c.Name == "company_id" && c.NotNull == 1 {
			needMigrate = true
			break
		}
	}
	if !needMigrate {
		return
	}
	log.Println("migrating contacts.company_id to nullable...")
	err := database.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`CREATE TABLE contacts_new (
			id integer PRIMARY KEY AUTOINCREMENT,
			company_id integer,
			name text NOT NULL,
			title text,
			phone text,
			email text,
			wechat text,
			notes text,
			created_at datetime,
			updated_at datetime
		)`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`INSERT INTO contacts_new (id, company_id, name, title, phone, email, wechat, notes, created_at, updated_at)
			SELECT id, NULLIF(company_id, 0), name, title, phone, email, wechat, notes, created_at, updated_at FROM contacts`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DROP TABLE contacts`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`ALTER TABLE contacts_new RENAME TO contacts`).Error; err != nil {
			return err
		}
		return tx.Exec(`CREATE INDEX IF NOT EXISTS idx_contacts_company_id ON contacts(company_id)`).Error
	})
	if err != nil {
		log.Printf("migrate contacts failed: %v", err)
	}
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

func startWALCheckpointLoop(database *gorm.DB) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		if err := database.Exec("PRAGMA wal_checkpoint(PASSIVE)").Error; err != nil {
			log.Printf("wal_checkpoint failed: %v", err)
		}
	}
}

func ensureDeletedAtColumn(database *gorm.DB, table string) {
	type colInfo struct {
		Name string `gorm:"column:name"`
	}
	var cols []colInfo
	if err := database.Raw("PRAGMA table_info(" + table + ")").Scan(&cols).Error; err != nil || len(cols) == 0 {
		return
	}
	for _, c := range cols {
		if c.Name == "deleted_at" {
			return
		}
	}
	_ = database.Exec("ALTER TABLE " + table + " ADD COLUMN deleted_at datetime").Error
	_ = database.Exec("CREATE INDEX IF NOT EXISTS idx_" + table + "_deleted_at ON " + table + "(deleted_at)").Error
}
