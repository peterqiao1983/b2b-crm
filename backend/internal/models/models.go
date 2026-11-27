package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	DisplayName  string    `gorm:"size:64" json:"display_name"`
	Role         string    `gorm:"size:32;default:user" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Company struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:200;not null;index" json:"name"`
	Stage     string         `gorm:"size:32;default:lead;index" json:"stage"`
	Industry  string         `gorm:"size:100" json:"industry"`
	Owner     string         `gorm:"size:64" json:"owner"`
	Source    string         `gorm:"size:100" json:"source"`
	Website   string         `gorm:"size:255" json:"website"`
	Address   string         `gorm:"size:255" json:"address"`
	Notes     string         `gorm:"type:text" json:"notes"`
	DevYear   int            `gorm:"default:0;index" json:"dev_year"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type Contact struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CompanyID *uint          `gorm:"index" json:"company_id"`
	Name      string         `gorm:"size:100;not null;index" json:"name"`
	Title     string         `gorm:"size:100" json:"title"`
	Phone     string         `gorm:"size:64" json:"phone"`
	Email     string         `gorm:"size:128" json:"email"`
	Wechat    string         `gorm:"size:64" json:"wechat"`
	Notes     string         `gorm:"type:text" json:"notes"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type Project struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	CompanyID       uint           `gorm:"not null;index" json:"company_id"`
	Name            string         `gorm:"size:200;not null;index" json:"name"`
	Stage           string         `gorm:"size:32;default:initial;index" json:"stage"`
	EstimatedAmount float64        `gorm:"default:0" json:"estimated_amount"`
	TargetDate      *time.Time     `json:"target_date"`
	Owner           string         `gorm:"size:64" json:"owner"`
	Description     string         `gorm:"type:text" json:"description"`
	DevYear         int            `gorm:"default:0;index" json:"dev_year"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type ProjectContact struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"not null;uniqueIndex:idx_project_contact" json:"project_id"`
	ContactID uint      `gorm:"not null;uniqueIndex:idx_project_contact" json:"contact_id"`
	Role      string    `gorm:"size:64" json:"role"`
	Attitude  string    `gorm:"size:32;default:neutral" json:"attitude"`
	Notes     string    `gorm:"type:text" json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

type FollowUp struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	CompanyID      uint           `gorm:"not null;index" json:"company_id"`
	ProjectID      *uint          `gorm:"index" json:"project_id"`
	ContactID      *uint          `gorm:"index" json:"contact_id"`
	FollowType     string         `gorm:"size:32;default:visit" json:"follow_type"`
	OccurredAt     time.Time      `gorm:"not null;index" json:"occurred_at"`
	NextFollowAt   *time.Time     `gorm:"index" json:"next_follow_at"`
	NextFollowNote string         `gorm:"size:255" json:"next_follow_note"`
	Summary        string         `gorm:"type:text;not null" json:"summary"`
	CreatedBy      string         `gorm:"size:64" json:"created_by"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

type AuditLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	Actor      string    `gorm:"size:64;not null" json:"actor"`
	Source     string    `gorm:"size:32;not null;index" json:"source"` // web, mcp
	Action     string    `gorm:"size:32;not null;index" json:"action"` // create, update, delete, restore
	EntityType string    `gorm:"size:32;not null;index" json:"entity_type"`
	EntityID   uint      `gorm:"index" json:"entity_id"`
	EntityName string    `gorm:"size:255" json:"entity_name"`
	Detail     string    `gorm:"type:text" json:"detail"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}
