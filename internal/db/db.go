package db

import (
	"fmt" // Added for fmt.Errorf
	"io"
	"os"
	"strconv"
	"strings"

	"fleet-management/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(
		&models.Car{},
		&models.Document{},
		&models.Repair{},
		&models.TireChange{},
		&models.Executor{},
		&models.DocumentType{},
		&models.Setting{},
	)
	if err != nil {
		return nil, err
	}

	DB = db

	// Seed default document types if none exist
	var dtCount int64
	db.Model(&models.DocumentType{}).Count(&dtCount)
	if dtCount == 0 {
		defaults := []models.DocumentType{
			{Name: "Страхування"},
			{Name: "Техпаспорт"},
			{Name: "Медична довідка"},
			{Name: "Інше"},
		}
		db.Create(&defaults)
	}

	return db, nil
}

// FindCarSmart searches for a car by full number or a suffix (e.g., last 4 digits).
func FindCarSmart(input string) (*models.Car, error) {
	var car models.Car
	input = strings.ToUpper(strings.TrimSpace(input))
	if err := DB.Where("number = ?", input).First(&car).Error; err == nil {
		return &car, nil
	}
	// Try partial match
	if err := DB.Where("number LIKE ?", "%"+input+"%").First(&car).Error; err == nil {
		return &car, nil
	}
	return nil, fmt.Errorf("car not found")
}

// NormalizeDescription removes multiple spaces, punctuation and lowercases the string
func NormalizeDescription(s string) string {
	s = strings.ToLower(s)
	// Replace punctuation with spaces
	replacer := strings.NewReplacer(",", " ", ".", " ", ";", " ", ":", " ", "(", " ", ")", " ", "-", " ", "_", " ")
	s = replacer.Replace(s)
	// Collapse multiple spaces
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

// GetSetting returns the value of a setting by key.
func GetSetting(key string, defaultValue string) string {
	var s models.Setting
	if err := DB.Where("key = ?", key).First(&s).Error; err != nil {
		return defaultValue
	}
	return s.Value
}

// SetSetting saves or updates a setting.
func SetSetting(key, value string) error {
	var s models.Setting
	err := DB.Where("key = ?", key).First(&s).Error
	if err != nil {
		s = models.Setting{Key: key, Value: value}
		return DB.Create(&s).Error
	}
	s.Value = value
	return DB.Save(&s).Error
}

// CloseDB closes the database connection
func CloseDB() error {
	if DB == nil {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ReplaceDB replaces the current database file with a new one
func ReplaceDB(newPath string) error {
	// 1. Close current connection
	if err := CloseDB(); err != nil {
		return fmt.Errorf("failed to close database: %v", err)
	}

	// 2. Backup current (optional but safe)
	_ = os.Rename("fleet.db", "fleet.db.bak")

	// 3. Move new file to fleet.db
	source, err := os.Open(newPath)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create("fleet.db")
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}

	// 4. Re-initialize
	_, err = InitDB("fleet.db")
	return err
}

// EvaluatePrice calculates the total from a simple expression like "2500+1500"
func EvaluatePrice(input string) (float64, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0, nil
	}
	// Support simple addition for now
	if strings.Contains(input, "+") {
		parts := strings.Split(input, "+")
		var total float64
		for _, part := range parts {
			val, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number: %s", part)
			}
			total += val
		}
		return total, nil
	}
	// Fallback to regular parse
	return strconv.ParseFloat(input, 64)
}
