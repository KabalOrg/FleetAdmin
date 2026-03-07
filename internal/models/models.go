package models

import (
	"time"

	"gorm.io/gorm"
)

// Car represents a vehicle in the fleet.
type Car struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Number    string         `gorm:"uniqueIndex;not null" json:"number"` // Full number, e.g., "ВІ5565СК"
	Model     string         `json:"model"`
	Year      int            `json:"year"`
	Owner     string         `json:"owner"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Documents   []Document   `json:"documents,omitempty"`
	Repairs     []Repair     `json:"repairs,omitempty"`
	TireChanges []TireChange `json:"tire_changes,omitempty"`
}

// Document represents a vehicle document with an expiry date.
type Document struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	CarID      uint           `json:"car_id"`
	CarNumber  string         `json:"car_number"`
	Type       string         `json:"type"`
	IssueDate  time.Time      `json:"issue_date"`
	ExpiryDate time.Time      `json:"expiry_date"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// Repair represents a maintenance record.
type Repair struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CarID       uint           `json:"car_id"`
	CarNumber   string         `json:"car_number"` // Stored full number
	Date        time.Time      `json:"date"`
	Mileage     int            `json:"mileage"`
	Description string         `json:"description"`
	Executor    string         `json:"executor"`
	Price       float64        `json:"price"`
	IsVAT       bool           `json:"is_vat"`
	Notes       string         `json:"notes"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TireChange represents a tire replacement record.
type TireChange struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CarID       uint           `json:"car_id"`
	CarNumber   string         `json:"car_number"`
	Date        time.Time      `json:"date"`
	Mileage     int            `json:"mileage"`
	Description string         `json:"description"`
	Executor    string         `json:"executor"`
	Price       float64        `json:"price"`
	IsVAT       bool           `json:"is_vat"`
	Notes       string         `json:"notes"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Executor represents a person or company that performs repairs.
type Executor struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex;not null" json:"name"`
	Phone     string         `json:"phone"`
	Notes     string         `json:"notes"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// DocumentType represents a configurable document type (e.g., "Страхування").
type DocumentType struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex;not null" json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Setting represents a key-value configuration.
type Setting struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Key       string         `gorm:"uniqueIndex;not null" json:"key"`
	Value     string         `json:"value"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
